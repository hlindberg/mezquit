package cmd

import (
	"encoding/csv"
	"fmt"
	"net"
	"os"

	"github.com/hlindberg/mezquit/internal/mqtt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "pub",
	Short: "Publish MQTT message",
	Long: `Publishes a message via MQTT

	`,
	Run: func(cmd *cobra.Command, args []string) {
		p := &publisher{}
		if TestQoS1Resend {
			p.qos1ResendPublish()
		} else {
			p.standardPublish()
		}
	},

	Args: func(cmd *cobra.Command, args []string) error {
		// Check any arguments
		if QoS < 0 || QoS > 2 {
			return fmt.Errorf("--qos must be between 0 and 2, got %d", QoS)
		}
		if KeepAliveSeconds < 0 {
			return fmt.Errorf("--keep_alive cannot be negative")
		}
		if TestQoS1Resend {
			if QoS != 1 {
				log.Debugf("QoS set to 1 since --test_qos1_resend was requested")
				QoS = 1
			}
		}
		return nil
	},
}

type publisher struct {
}

func (p *publisher) dial() net.Conn {
	// This MQTT client uses a hard coded broker and unencrypted TCP port
	// It gives the resulting conn as both Input and Output to a MQTT Session
	//
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", MQTTBroker, mqtt.UnencryptedPortTCP))
	if err != nil {
		panic(err)
	}
	return conn
}

func (p *publisher) clientName() string {
	if MQTTClientName == "" {
		MQTTClientName = mqtt.RandomClientID()
		log.Infof("Using generated client ID %s", MQTTClientName)
	}
	return MQTTClientName
}

func (p *publisher) session(clientName string, conn net.Conn) *mqtt.Session {
	return mqtt.NewSession(mqtt.ClientID(clientName), mqtt.InputOutput(conn))
}

func (p *publisher) connect(session *mqtt.Session, ignorePubAck bool, cleanSession bool) {
	err := session.Connect(
		mqtt.WillTopic(WillTopic),
		mqtt.WillMessage([]byte(WillMessage)),
		mqtt.WillQoS(WillQoS),
		mqtt.WillRetain(WillRetain),
		mqtt.KeepAliveSeconds(KeepAliveSeconds),
		mqtt.XIgnorePubAck(ignorePubAck),
		mqtt.CleanSession(cleanSession),
	)

	if err != nil {
		panic(err)
	}
}

func (p *publisher) publishMessage(session *mqtt.Session) {
	session.Publish(mqtt.Message([]byte(Message)),
		mqtt.Topic(Topic),
		mqtt.QoS(QoS),
		mqtt.Retain(Retain),
	)
}

func (p *publisher) publishFromFile(session *mqtt.Session) {
	f, err := os.Open(FileName)
	if err != nil {
		panic(fmt.Sprintf("Cannot open file %s", FileName))
	}
	all, err := csv.NewReader(f).ReadAll()
	for _, r := range all {
		session.Publish(mqtt.Message([]byte(r[1])),
			mqtt.Topic(r[0]),
			mqtt.QoS(QoS),
			mqtt.Retain(false),
		)
	}
}

func (p *publisher) publishGivenMessage(session *mqtt.Session) {
	if FileName == "" {
		p.publishMessage(session)
	} else {
		p.publishFromFile(session)
	}
}

func (p *publisher) standardPublish() {
	conn := p.dial()
	clientName := p.clientName()
	session := p.session(clientName, conn)
	p.connect(session, false, true)
	p.publishGivenMessage(session)

	if TestNoDisconnect {
		session.DisconnectWithoutMessage(1)
	} else {
		session.Disconnect(1)
	}
	// Done
	conn.Close()
}

func (p *publisher) qos1ResendPublish() {
	// First pass where PUBACK is ignored
	conn := p.dial()
	clientName := p.clientName()
	session := p.session(clientName, conn)
	p.connect(session, true, true) // ignore PUBACK, clean session
	p.publishGivenMessage(session)

	if TestNoDisconnect {
		session.DisconnectWithoutMessage(1)
	} else {
		session.Disconnect(1)
	}
	// Done
	conn.Close()

	// Second Pass
	conn = p.dial()
	// Set new input/output to second connect
	session.ReEstablish(mqtt.InputOutput(conn))
	p.connect(session, false, false) // process PUBACK, not clean session

	if TestNoDisconnect {
		session.DisconnectWithoutMessage(1)
	} else {
		session.Disconnect(1)
	}
	// Done
	conn.Close()

}

// MQTTBroker is the MQTT host:port to dial
var MQTTBroker string

// MQTTClientName is the MQTT client name - a short UUID by default
var MQTTClientName string

// Topic is the MQTT topic to publish to
var Topic string

// Message is the MQTT message text to publish
var Message string

// KeepAliveSeconds is the MQTT number of seconds to keep a connection alive
var KeepAliveSeconds int

// QoS is the MQQT quality of service to publish at (and also to connect with)
var QoS int

// FileName the name of a file to read instead of using --topic and --message
var FileName string

// Retain indicates if the published message should be retained
var Retain bool

// WillMessage is the MQTT message text to send on a dirty disconnect
var WillMessage string

// WillTopic is the MQTT message text to send on a dirty disconnect
var WillTopic string

// WillQoS is the QoS for the delivery of the WILL message
var WillQoS int

// WillRetain is the retain flag for the WILL message publishing
var WillRetain bool

// TestNoDisconnect if true no DISCONNECT is sent thereby allowing WILL features to be tested
var TestNoDisconnect bool

// TestQoS1Resend if true no DISCONNECT is sent thereby allowing WILL features to be tested
var TestQoS1Resend bool

func init() {
	RootCmd.AddCommand(publishCmd)
	flags := publishCmd.PersistentFlags()

	flags.StringVarP(&MQTTBroker,
		"broker", "b", "localhost", "the MQTT Broker host to connect to (default 'localhost')")
	flags.StringVarP(&MQTTClientName,
		"client", "c", "", "the MQTT client name to use - default is a short UUID")
	flags.StringVarP(&FileName,
		"file", "f", "", "File with CSV <topic, message> lines to publish")
	flags.IntVarP(&KeepAliveSeconds,
		"keep_alive", "", 0, "sets the number of seconds to keep a connection alive")
	flags.StringVarP(&Message,
		"message", "m", "", "the message to send")
	flags.StringVarP(&Topic,
		"topic", "t", "test", "the MQTT topic to send message to (default 'test')")
	flags.IntVarP(&QoS,
		"qos", "q", 0, "Quality of service 0-2 (default 0)")
	flags.BoolVarP(&Retain,
		"retain", "r", false, "If message should be retained")
	flags.StringVarP(&WillMessage,
		"wmessage", "", "", "the will message to send when disconnect is not clean")
	flags.IntVarP(&WillQoS,
		"wqos", "", 0, "Quality of service 0-2 (default 0) for publishing of WILL message")
	flags.BoolVarP(&WillRetain,
		"wretain", "", false, "If WILL message should be retained")
	flags.StringVarP(&WillTopic,
		"wtopic", "", "", "the topic for a will message to send when disconnect is not clean")

	// Options for testing unclean operations
	flags.BoolVarP(&TestNoDisconnect,
		"test_no_disconnect", "", false, "do not send DISCONNECT to test WILL features")
	flags.BoolVarP(&TestQoS1Resend,
		"test_qos1_resend", "", false, "Performs: CONNECT, send message(s), ignore PUBACK(s), DISCONNECT, CONNECT with clean=false, resend, DISCONNECT")

}
