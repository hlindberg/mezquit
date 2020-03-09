package logging

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// SetLevelFromName sets the logging level based on a string level name
func SetLevelFromName(levelName string) {
	// Only log the warning severity or above.
	level, err := log.ParseLevel(levelName)
	if err != nil {
		// tricky situation... should have been handled when validating user input
		log.SetLevel(log.WarnLevel)
		log.Warn(fmt.Sprintf("Unknown loglevel '%s' - using loglevel=warn", levelName))
		return
	}
	log.SetLevel(level)
	log.Info(fmt.Sprintf("Loglevel set to %s", levelName))
}
