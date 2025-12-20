package main

import (
	"errors"
	"flag"
	"os"
	"time"
)

// Config holds all configuration for the ddns-client.
type Config struct {
	// Required
	APIKey   string
	Owner    string
	Location string

	// Optional with defaults
	APIURL   string
	StateDir string
	Interval time.Duration
	IPv6     bool
	Verbose  bool
	Cron     bool
}

// DefaultConfig returns configuration with default values.
func DefaultConfig() Config {
	return Config{
		APIURL:   "https://ddns.grocky.net",
		Interval: 15 * time.Minute,
	}
}

// LoadConfig loads configuration from environment variables and flags.
// Flags take precedence over environment variables.
func LoadConfig() (Config, error) {
	cfg := DefaultConfig()

	// Define flags
	apiKey := flag.String("api-key", "", "API key for authentication")
	owner := flag.String("owner", "", "Owner ID")
	location := flag.String("location", "", "Location name")
	apiURL := flag.String("api-url", cfg.APIURL, "DDNS API URL")
	stateDir := flag.String("state-dir", "", "State directory (default: ~/.config/ddns-client/)")
	interval := flag.Duration("interval", cfg.Interval, "Check interval for daemon mode")
	ipv6 := flag.Bool("6", false, "Use IPv6 instead of IPv4")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	cron := flag.Bool("cron", false, "Run once and exit (for crontab)")

	flag.Parse()

	// Load from environment (lower priority)
	cfg.APIKey = os.Getenv("DDNS_API_KEY")
	cfg.Owner = os.Getenv("DDNS_OWNER")
	cfg.Location = os.Getenv("DDNS_LOCATION")

	// Override with flags (higher priority)
	if *apiKey != "" {
		cfg.APIKey = *apiKey
	}
	if *owner != "" {
		cfg.Owner = *owner
	}
	if *location != "" {
		cfg.Location = *location
	}

	cfg.APIURL = *apiURL
	cfg.StateDir = *stateDir
	cfg.Interval = *interval
	cfg.IPv6 = *ipv6
	cfg.Verbose = *verbose
	cfg.Cron = *cron

	return cfg, nil
}

// Validate checks that required configuration is present.
func (c Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("API key is required (set DDNS_API_KEY or use --api-key)")
	}
	if c.Owner == "" {
		return errors.New("owner is required (set DDNS_OWNER or use --owner)")
	}
	if c.Location == "" {
		return errors.New("location is required (set DDNS_LOCATION or use --location)")
	}
	return nil
}
