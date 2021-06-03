package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/everactive/iot-agent/pkg/config"
	"github.com/everactive/iot-agent/pkg/server"
	"github.com/spf13/cobra"
)

var Agent = &cobra.Command{
	Use:   "agent",
	Short: "agent",
	Long:  "agent",
	Run: func(cmd *cobra.Command, args []string) {
		log.SetLevel(log.TraceLevel)
		log.Println("Server starting")
		logLevel := os.Getenv("LOG_LEVEL")
		log.Infof("env LOG_LEVEL: %s", logLevel)
		if len(logLevel) > 0 {
			l, err := log.ParseLevel(logLevel)
			if err != nil {
				log.SetLevel(log.TraceLevel)
				log.Tracef("LOG_LEVEL environment variable is set to %s, could not parse to a valid log level. Using trace logging.", logLevel)
			} else {
				log.SetLevel(l)
			}
			log.Infof("Using LOG_LEVEL %s", log.GetLevel())
		} else {
			log.Infof("LOG_LEVEL not set by environment, using default TRACE.")
		}

		log.Println("Starting IoT agent")
		config.InitializeConfig()
		serverInstance := server.New()
		serverInstance.Run()
	},
}
