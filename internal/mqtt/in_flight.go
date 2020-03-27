package mqtt

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	// NUM64BITS is the number of entries in the in flight window
	NUM64BITS = 1024
)

type inFlight struct {
	bits        [NUM64BITS]uint64 // bitset for packet ID 1 - 0xFFFF (= 8k data)
	mutex       *sync.Mutex
	nextValue   int
	waitingIdx  map[int]*waitingPacket
	waitingList *waitingPacketList
}

// newInFlight produces a new InFlight session state for packet IDs and packets in Flight
//
func newInFlight() *inFlight {
	f := &inFlight{mutex: &sync.Mutex{}, nextValue: 0, waitingList: &waitingPacketList{}, waitingIdx: make(map[int]*waitingPacket)}
	return f
}

func (f *inFlight) setBit(n int) {
	nbyte := n / 64
	nbit := n % 64
	f.bits[nbyte] |= 1 << nbit
}

func (f *inFlight) unsetBit(n int) {
	nbyte := n / 64
	nbit := n % 64
	f.bits[nbyte] &= ^(1 << nbit)
}

func (f *inFlight) getBit(n int) bool {
	nbyte := n / 64
	nbit := n % 64
	return (f.bits[nbyte] & (1 << nbit)) > 0
}

func (f *inFlight) clearAllBits() {
	for i := 0; i < NUM64BITS; i++ {
		f.bits[i] = 0
	}
}

func (f *inFlight) nextPacketID() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	dragonsTail := f.nextValue
	result := cappedIncrement(f.nextValue)

	// Make sure value is available - loop until one is found if needed
	// If none is found there are no more outstanding values to be read and the sender must wait
	// until some become available - alternatively error
	//
	for ; f.getBit(result) && result != dragonsTail; result = cappedIncrement(result) {
	}
	if result == dragonsTail {
		// Not good - all are taken
		// TODO: need solution what to do here
		panic("No free packet IDs")
	}
	f.nextValue = result
	f.setBit(result) // Set the returned value as "in flight"
	return result
}

func cappedIncrement(x int) int {
	x++
	if x > 0xFFFF {
		x = 1
	}
	return x
}

type waitingPacket struct {
	msg      MessageWriter
	packetID int
	next     *waitingPacket
	prev     *waitingPacket
}

func (wp *waitingPacket) nextPacket() *waitingPacket {
	return wp.next
}

func (wp *waitingPacket) prevPacket() *waitingPacket {
	return wp.prev
}

// registerWaiting registers a package with a given packetID as waiting for an ACK of some kind
func (f *inFlight) registerWaiting(packetID int, msg MessageWriter) {
	// TODO: Could use separate mutex
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.setBit(packetID) // Just in case, and this also allows test to just register packets will self asserted unique values
	theElement := f.waitingList.PushBack(&waitingPacket{msg: msg, packetID: packetID})
	f.waitingIdx[packetID] = theElement
}

// relaseWaiting drops the packet with the given packetID from the ordered set of packets waiting for ACK, but does not free the ID
// as there may be a new packet (QoS==2) of different kind for the same package ID
//
func (f *inFlight) releaseWaiting(packetID int) {
	log.Debugf("releaseWaitingPacket(%d)", packetID)
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if !f.getBit(packetID) {
		panic("Cannot release a packet that is not registered as waiting")
	}
	theElement := f.waitingIdx[packetID]
	if theElement == nil {
		log.Debugf("no registered packed for packed ID: %d", packetID)
	}
	f.waitingList.Remove(theElement)
}

// replaceWaiting replaces the message for the given packetID
func (f *inFlight) replaceWaiting(packetID int, msg MessageWriter) {
	log.Debugf("replaceWaiting(%d)", packetID)
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if !f.getBit(packetID) {
		panic("Cannot replace a packet that is not registered as waiting")
	}
	theElement := f.waitingIdx[packetID]
	if theElement == nil {
		log.Debugf("no registered packed for packed ID: %d", packetID)
	}
	// Replace
	theElement.msg = msg
}

// eachWaitingPackage yields each packet to the given function - the intent is for a caller
// to perfom re-sending of the bits after having marked the send as a duplicate
// A lock is kept until the entire iteration is done (or there is a panic) - thus, it is not allowed
// to claim any new IDs or register any new packets during this iteration.
//
func (f *inFlight) eachWaitingPacket(lambda func(packetID int, msg MessageWriter)) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	for e := f.waitingList.Front(); e != nil; e = e.nextPacket() {
		lambda(e.packetID, e.msg)
	}
}

type waitingPacketList struct {
	front *waitingPacket
	back  *waitingPacket
}

func (wp *waitingPacketList) Front() *waitingPacket {
	return wp.front
}
func (wp *waitingPacketList) Back() *waitingPacket {
	return wp.back
}
func (wp *waitingPacketList) PushBack(it *waitingPacket) *waitingPacket {
	if it == nil {
		panic("nil entry is not allowed in waitingPacketList")
	}
	if wp.front == nil {
		wp.front = it
		wp.back = it
		it.next = nil
		it.prev = nil
	} else {
		it.prev = wp.back
		it.next = nil
		wp.back.next = it
		wp.back = it
	}
	return it
}

func (wp *waitingPacketList) Remove(it *waitingPacket) {
	tmpNext := it.next
	tmpPrev := it.prev
	if tmpNext == nil {
		// has no next, must be last
		wp.back = tmpPrev
	} else {
		// has a next, needs to make it.prev it's prev
		it.next.prev = tmpPrev
	}
	if it.prev == nil {
		// has no prev, must be front
		wp.front = tmpNext
	} else {
		it.prev.next = tmpNext
	}
	// unlink it
	it.next = nil
	it.prev = nil
}
