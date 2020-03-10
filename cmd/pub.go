package cmd

import (
	"github.com/hlindberg/mezquit/internal/mqtt"
	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "pub",
	Short: "Publish MQTT message",
	Long: `Publishes a message via MQTT

	`,
	Run: func(cmd *cobra.Command, args []string) {
		// Call the busn logic
		mqtt.Publish(MQQTBroker, "dummytopic", "dummy message")
	},

	Args: func(cmd *cobra.Command, args []string) error {
		// Check any arguments
		return nil
	},
}

// MQQTBroker is the MQTT host:port to dial
var MQQTBroker string

func init() {
	RootCmd.AddCommand(publishCmd)
	flags := publishCmd.PersistentFlags()

	flags.StringVarP(&MQQTBroker, "broker", "b", "mqtt.eclipse.org", "the MQTT Broker host to connect to (IP or name)")
}
