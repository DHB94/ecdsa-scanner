package config

import (
	"os"
)

// Config holds all application configuration
type Config struct {
	DatabaseURL      string
	InfuraAPIKey     string
	Port             string
	BindAddrs        string
	PushoverAppToken string
	PushoverUserKey  string
}

// ChainConfig defines a blockchain to scan
type ChainConfig struct {
	Name        string
	ChainID     int
	RPCURL      string
	ExplorerURL string
	Enabled     bool
}

// ChainByID returns chain config by ID
func ChainByID(id int) *ChainConfig {
	for _, c := range DefaultChains() {
		if c.ChainID == id {
			return &c
		}
	}
	return nil
}

// ChainByName returns chain config by name
func ChainByName(name string) *ChainConfig {
	for _, c := range DefaultChains() {
		if c.Name == name {
			return &c
		}
	}
	return nil
}

// Load reads configuration from environment variables
func Load() *Config {
	cfg := &Config{
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		InfuraAPIKey:     os.Getenv("INFURA_API_KEY"),
		Port:             os.Getenv("PORT"),
		BindAddrs:        os.Getenv("BIND_ADDRS"),
		PushoverAppToken: os.Getenv("PUSHOVER_APP_TOKEN"),
		PushoverUserKey:  os.Getenv("PUSHOVER_USER_KEY"),
	}

	if cfg.Port == "" {
		cfg.Port = "8000"
	}
	if cfg.BindAddrs == "" {
		cfg.BindAddrs = "0.0.0.0"
	}

	return cfg
}

// DefaultChains returns the list of chains to scan
func DefaultChains() []ChainConfig {
	return []ChainConfig{
		{Name: "Ethereum", ChainID: 1, RPCURL: "https://mainnet.infura.io/v3", ExplorerURL: "https://etherscan.io", Enabled: true},
		{Name: "Polygon", ChainID: 137, RPCURL: "https://polygon-mainnet.infura.io/v3", ExplorerURL: "https://polygonscan.com", Enabled: true},
		{Name: "Arbitrum", ChainID: 42161, RPCURL: "https://arbitrum-mainnet.infura.io/v3", ExplorerURL: "https://arbiscan.io", Enabled: true},
		{Name: "Avalanche", ChainID: 43114, RPCURL: "https://avalanche-mainnet.infura.io/v3", ExplorerURL: "https://snowtrace.io", Enabled: true},
		{Name: "Optimism", ChainID: 10, RPCURL: "https://optimism-mainnet.infura.io/v3", ExplorerURL: "https://optimistic.etherscan.io", Enabled: true},
		{Name: "Base", ChainID: 8453, RPCURL: "https://base-mainnet.infura.io/v3", ExplorerURL: "https://basescan.org", Enabled: true},
		{Name: "Linea", ChainID: 59144, RPCURL: "https://linea-mainnet.infura.io/v3", ExplorerURL: "https://lineascan.build", Enabled: true},
	}
}

// SystemAddresses returns addresses to filter out (L2 system transactions)
func SystemAddresses() map[string]bool {
	return map[string]bool{
		"0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001": true, // Optimism/Base L1 deposits
		"0x00000000000000000000000000000000000a4b05": true, // Arbitrum system
		"0x977f82a600a1414e583f7f13623f1ac5d58b1c0b": true, // Celo system
	}
}
