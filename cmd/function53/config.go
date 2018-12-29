package main

import (
	"github.com/function61/gokit/jsonfile"
	"github.com/spf13/cobra"
)

const (
	configFilePath = "config.json"
)

type Config struct {
	MetricsPort int              `json:"metrics_port"`
	Endpoints   []ServerEndpoint `json:"endpoints"`
}

func defaultConfig() Config {
	return Config{
		MetricsPort: 9094,
		Endpoints: []ServerEndpoint{
			// 60 second inactivity timeout (not even TCP keepalive fixes this)
			// {"dns.google", "8.8.8.8:853"},
			// {"dns.google", "8.8.4.4:853"},

			// 10 second inactivity timeout (not even TCP keepalive fixes this)
			{"cloudflare-dns.com", "1.1.1.1:853"},
			{"cloudflare-dns.com", "1.0.0.1:853"},
		},
	}
}

func readConfig() (*Config, error) {
	conf := &Config{}
	return conf, jsonfile.Read(configFilePath, conf, true)
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
			if err := writeConfig(defaultConfig()); err != nil {
				panic(err)
			}
		},
	}
}
