package cmd

import (
	"fmt"
	"net"

	"github.com/hlindberg/mezquit/internal/mqtt"
	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "pub",
	Short: "Publish MQTT message",
	Long: `Publishes a message via MQTT

	`,
	Run: func(cmd *cobra.Command, args []string) {

		// This MQTT client uses a hard coded broker and unencrypted TCP port
		// It gives the resulting conn as both Input and Output to a MQTT Session
		//
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", MQQTBroker, mqtt.UnencryptedPortTCP))
		if err != nil {
			panic(err)
		}

		session := mqtt.NewSession(mqtt.ClientID("H3LLTest"), mqtt.InputOutput(conn))
		err = session.Connect()
		if err != nil {
			panic(err)
		}

		session.Publish(mqtt.Message([]byte(Message)),
			mqtt.Topic(Topic),
			mqtt.QoS(0),
			mqtt.Retain(false),
		)
		session.Disconnect(true, 0)
		// Done
		conn.Close()
	},

	Args: func(cmd *cobra.Command, args []string) error {
		// Check any arguments
		return nil
	},
}

// MQQTBroker is the MQTT host:port to dial
var MQQTBroker string

// Topic is the MQTT topic to publish to
var Topic string

// Message is the MQTT message text to publish
var Message string

func init() {
	RootCmd.AddCommand(publishCmd)
	flags := publishCmd.PersistentFlags()

	flags.StringVarP(&MQQTBroker, "broker", "b", "mqtt.eclipse.org", "the MQTT Broker host to connect to (default 'mgtt.eclipse.org')")
	flags.StringVarP(&Topic, "topic", "t", "test", "the MQTT topic to send message to (default 'test')")
	flags.StringVarP(&Message, "message", "m", "", "the message to send")
}
