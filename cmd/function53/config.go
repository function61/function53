package main

import (
	"github.com/function61/gokit/encoding/jsonfile"
	"github.com/function61/gokit/os/osutil"
	"github.com/spf13/cobra"
)

const (
	configFilePath = "config.json"
)

type Config struct {
	MetricsPort              int                           `json:"metrics_port"`
	Endpoints                []ServerEndpoint              `json:"dns_servers"`                         // DNS servers to use
	BlocklistDisableUpdates  bool                          `json:"blocklist_disable_updates,omitempty"` // if you need, you can disable updating the blocklist
	LogQueries               bool                          `json:"log_queries"`                         // whether to log DNS queries
	DefaultOverridableConfig *OverridableConfig            `json:"default_client_config"`               // applies to all client DNS queries, unless more specific entry in overrides_by_client_addr
	OverridesByClientAddr    map[string]*OverridableConfig `json:"overrides_by_client_addr"`            // you can disable internet OR ad/malware blocking per device
}

type OverridableConfig struct {
	RejectAllQueries    bool `json:"reject_all_queries"` // an easy (but not most secure) way to keep an IoT device off from internet
	DisableBlocklisting bool `json:"disable_blocking"`   // disable ad, malware etc. blocking
}

func defaultConfig() Config {
	return Config{
		MetricsPort: 9090,
		Endpoints: []ServerEndpoint{
			// 60 second inactivity timeout (not even TCP keepalive fixes this)
			// {"dns.google", "8.8.8.8:853"},
			// {"dns.google", "8.8.4.4:853"},

			// 10 second inactivity timeout (not even TCP keepalive fixes this)
			{"cloudflare-dns.com", "1.1.1.1:853"},
			{"cloudflare-dns.com", "1.0.0.1:853"},
		},
		LogQueries: true,

		DefaultOverridableConfig: &OverridableConfig{},

		OverridesByClientAddr: map[string]*OverridableConfig{},
	}
}

func readConfig() (*Config, error) {
	conf := &Config{}
	if err := jsonfile.Read(configFilePath, conf, true); err != nil {
		return nil, err
	}

	// make it so that rest of code can assume that this is present
	if conf.DefaultOverridableConfig == nil {
		conf.DefaultOverridableConfig = &OverridableConfig{}
	}

	return conf, nil
}

func writeConfig(conf Config) error {
	return jsonfile.Write(configFilePath, &conf)
}

func writeDefaultConfigEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "write-default-config",
		Short: "Writes default config file",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(writeConfig(defaultConfig()))
		},
	}
}
