package cmd

import (
	"fmt"
	"github.com/everactive/iot-agent/config"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	Configure.Flags().StringP("url", "u", "http://localhost:8030", "Specify the URL of the identity service")
}

var Configure = &cobra.Command{
	Use:   "configure",
	Short: "configure",
	Long:  "configure",
	Run: func(cmd *cobra.Command, args []string) {
		url, err := cmd.Flags().GetString("url")
		if err != nil {
			fmt.Printf("Error getting url flag: %s\n", err)
		}
		if url != "" {
			fmt.Printf("Using url %s\n", url)
			// Store the URL (let the other parameters be defaulted)
			err := config.StoreParameters(config.Settings{
				IdentityURL: url,
			})
			if err != nil {
				fmt.Println("Error saving parameters:", err)
				os.Exit(1)
			}
		}
	},
}
