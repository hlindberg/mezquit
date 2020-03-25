package mqtt

import (
	"testing"

	"github.com/hlindberg/mezquit/testutils"
)

func Test_CanCreateNewInFlight_AndGetPacketID_1(t *testing.T) {
	inF := newInFlight()
	next := inF.nextPacketID()
	testutils.CheckEqual(1, next, t)
}

func Test_InFligth_Produces_values_1_to_0XFFF(t *testing.T) {
	inF := newInFlight()
	for i := 1; i <= 0XFFFF; i++ {
		testutils.CheckEqual(i, inF.nextPacketID(), t)
	}
}

func Test_InFligth_Produces_values_1_to_0XFFF_and_flips_to_1(t *testing.T) {
	inF := newInFlight()
	for i := 1; i <= 0XFFFF; i++ {
		testutils.CheckEqual(i, inF.nextPacketID(), t)
	}
	inF.clearAllBits() // Without this all bits would be taken and there would be a panic
	testutils.CheckEqual(1, inF.nextPacketID(), t)
}

func Test_InFligth_skips_claimed_IDs_when_producing_next_packet_id(t *testing.T) {
	inF := newInFlight()
	inF.setBit(1)
	inF.setBit(2)
	//
	inF.setBit(4)
	//
	inF.setBit(6)
	inF.setBit(7)
	//
	testutils.CheckEqual(3, inF.nextPacketID(), t)
	testutils.CheckEqual(5, inF.nextPacketID(), t)
	testutils.CheckEqual(8, inF.nextPacketID(), t)
}

func Test_InFligth_unsetBit_makes_ID_available_as_next_packet_id(t *testing.T) {
	inF := newInFlight()
	inF.setBit(1)
	inF.setBit(2)
	inF.setBit(3)
	inF.setBit(4)
	//
	inF.unsetBit(3)
	testutils.CheckEqual(3, inF.nextPacketID(), t)
}

func Test_waitingPacketList_can_be_instantiated_and_is_then_empty(t *testing.T) {
	wpl := waitingPacketList{}
	testutils.CheckNil(wpl.Front(), t)
	testutils.CheckNil(wpl.Back(), t)
}
func Test_waitingPacketList_accepts_addition_of_waitingPacket_and_it_becomes_both_Front_and_Back(t *testing.T) {
	wpl := waitingPacketList{}
	it := waitingPacket{}

	wpl.PushBack(&it)
	testutils.CheckEqual(&it, wpl.Front(), t)
	testutils.CheckEqual(&it, wpl.Back(), t)
	testutils.CheckNil(it.nextPacket(), t)
	testutils.CheckNil(it.prevPacket(), t)
}

func Test_waitingPacketList_accepts_addition_of_subsequent_waitingPacket_becomes_Back(t *testing.T) {
	wpl := waitingPacketList{}
	it := waitingPacket{}
	it2 := waitingPacket{}

	wpl.PushBack(&it)
	wpl.PushBack(&it2)
	testutils.CheckEqual(&it, wpl.Front(), t)
	testutils.CheckEqual(&it2, wpl.Back(), t)

	testutils.CheckEqual(&it2, it.nextPacket(), t)
	testutils.CheckNil(it.prevPacket(), t)

	testutils.CheckNil(it2.nextPacket(), t)
	testutils.CheckEqual(&it, it2.prevPacket(), t)
}

func Test_waitingPacketList_a_Remove_of_single_waitingPackage_makes_list_empty(t *testing.T) {
	wpl := waitingPacketList{}
	it := waitingPacket{}

	wpl.PushBack(&it)
	wpl.Remove(&it)
	testutils.CheckNil(wpl.Front(), t)
	testutils.CheckNil(wpl.Back(), t)
}

func Test_waitingPacketList_a_Remove_of_multiple_waitingPackage_closes_each_gap_when_doing_middle_last_remove(t *testing.T) {
	wpl := waitingPacketList{}
	it1 := waitingPacket{}
	it2 := waitingPacket{}
	it3 := waitingPacket{}

	wpl.PushBack(&it1)
	wpl.PushBack(&it2)
	wpl.PushBack(&it3)

	wpl.Remove(&it2)
	testutils.CheckEqual(&it3, it1.nextPacket(), t) // gap closed
	testutils.CheckEqual(&it1, it3.prevPacket(), t) // gap closed
	testutils.CheckNil(it2.nextPacket(), t)         // unlinked
	testutils.CheckNil(it2.prevPacket(), t)         // unlinked

	wpl.Remove(&it3)
	testutils.CheckEqual(&it1, wpl.Front(), t)
	testutils.CheckEqual(&it1, wpl.Back(), t)
	testutils.CheckNil(it1.nextPacket(), t) // unlinked
	testutils.CheckNil(it1.prevPacket(), t) // unlinked
	testutils.CheckNil(it3.nextPacket(), t) // unlinked
	testutils.CheckNil(it3.prevPacket(), t) // unlinked
}

func Test_waitingPacketList_a_Remove_of_multiple_waitingPackage_closes_each_gap_when_doing_middle_first_remove(t *testing.T) {
	wpl := waitingPacketList{}
	it1 := waitingPacket{}
	it2 := waitingPacket{}
	it3 := waitingPacket{}

	wpl.PushBack(&it1)
	wpl.PushBack(&it2)
	wpl.PushBack(&it3)

	wpl.Remove(&it2)
	testutils.CheckEqual(&it3, it1.nextPacket(), t) // gap closed
	testutils.CheckEqual(&it1, it3.prevPacket(), t) // gap closed
	testutils.CheckNil(it2.nextPacket(), t)         // unlinked
	testutils.CheckNil(it2.prevPacket(), t)         // unlinked

	wpl.Remove(&it1)
	testutils.CheckEqual(&it3, wpl.Front(), t)
	testutils.CheckEqual(&it3, wpl.Back(), t)
	testutils.CheckNil(it1.nextPacket(), t) // unlinked
	testutils.CheckNil(it1.prevPacket(), t) // unlinked
	testutils.CheckNil(it3.nextPacket(), t) // unlinked
	testutils.CheckNil(it3.prevPacket(), t) // unlinked
}

func Test_waitingPacketList_does_not_accept_PushBack_of_nil(t *testing.T) {
	wpl := waitingPacketList{}
	defer testutils.ShouldPanic(t)
	wpl.PushBack(nil)
}

func Test_inFlight_eachWaitingPacket_yields_each_waiting_package(t *testing.T) {
	inF := newInFlight()
	data1 := GenericMessage{fixedHeader: 0, body: []byte{7}}
	data2 := GenericMessage{fixedHeader: 0, body: []byte{8}}
	data3 := GenericMessage{fixedHeader: 0, body: []byte{9}}
	inF.registerWaiting(1, &data1)
	inF.registerWaiting(2, &data2)
	inF.registerWaiting(3, &data3)
	val := 0
	inF.eachWaitingPacket(func(id int, msg MessageWriter) {
		val += id
	})
	testutils.CheckEqual(1+2+3, val, t)
}

func Test_inFlight_releaseWaitingPacket_drops_it_from_list_without_resetting_bit(t *testing.T) {
	inF := newInFlight()
	inF.setBit(1)
	inF.setBit(2)
	inF.setBit(3)
	data1 := GenericMessage{fixedHeader: 0, body: []byte{7}}
	data2 := GenericMessage{fixedHeader: 0, body: []byte{8}}
	data3 := GenericMessage{fixedHeader: 0, body: []byte{9}}
	inF.registerWaiting(1, &data1)
	inF.registerWaiting(2, &data2)
	inF.registerWaiting(3, &data3)

	inF.releaseWaiting(3)

	val := 0
	inF.eachWaitingPacket(func(id int, data MessageWriter) {
		val += id
	})
	testutils.CheckEqual(1+2, val, t)
	testutils.CheckTrue(inF.getBit(3), t)
}

func Test_inFlight_releaseWaitingPacket_panics_on_non_registered_package(t *testing.T) {
	inF := newInFlight()
	defer testutils.ShouldPanic(t)
	inF.releaseWaiting(1)
}

func Test_cappedIncrement_caps_increment_at_0xFFFF_flips_to_1(t *testing.T) {
	x := 0xFFFF
	testutils.CheckEqual(1, cappedIncrement(x), t)
}
