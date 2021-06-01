package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

const envPrefix = "iotagent"

// Configuration keys
const (
	NATSConnectionRetryIntervalKey = "nats.connection.retry.interval"
	NATSSnapdPasswordKey           = "nats.snapd.password"
)

// nolint:mnd
var DefaultConfig = map[string]interface{}{
	NATSConnectionRetryIntervalKey: 10 * time.Second,
	// NATSSnapdPassword defaults to unset
}

func InitializeConfig() {
	// Load default, non-struct config values
	for k, v := range DefaultConfig {
		viper.SetDefault(k, v)
	}

	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
}
