package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grocky/ddns-service/internal/client"
	"github.com/grocky/ddns-service/internal/state"
	"github.com/grocky/ddns-service/pkg/pubip"
)

func main() {
	setupProfiling()
	defer stopProfiling()

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Set up logger
	logLevel := slog.LevelInfo
	if cfg.Verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	if cfg.Cron {
		if err := runCron(cfg, logger); err != nil {
			logger.Error("update failed", "error", err)
			os.Exit(1)
		}
	} else {
		if err := runDaemon(cfg, logger); err != nil {
			logger.Error("daemon failed", "error", err)
			os.Exit(1)
		}
	}
}

func init() {
	flag.Usage = func() {
		fmt.Println(`ddns-client - Dynamic DNS update client

Usage:
  ddns-client [flags]

By default, runs as a daemon checking for IP changes periodically.
Use --cron for one-shot mode (suitable for crontab).

Environment Variables:
  DDNS_API_KEY    API key for authentication (preferred over --api-key)
  DDNS_OWNER      Owner ID
  DDNS_LOCATION   Location name

Flags:`)
		flag.PrintDefaults()
		fmt.Println(`
Examples:
  # Daemon mode (default)
  export DDNS_API_KEY=ddns_sk_...
  export DDNS_OWNER=myuser
  export DDNS_LOCATION=home
  ddns-client

  # Cron mode
  ddns-client --cron

  # IPv6 mode with verbose logging
  ddns-client -6 --verbose`)
	}
}

// runDaemon runs the client in continuous daemon mode.
func runDaemon(cfg Config, logger *slog.Logger) error {
	logger.Info("starting daemon mode",
		"interval", cfg.Interval,
		"owner", cfg.Owner,
		"location", cfg.Location,
		"ipv6", cfg.IPv6,
	)

	// Create API client
	apiClient := client.New(client.Config{
		APIURL: cfg.APIURL,
		APIKey: cfg.APIKey,
	})

	// Track last known IP in memory
	var lastIP string

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initial check
	if err := checkAndUpdate(ctx, cfg, apiClient, &lastIP, logger); err != nil {
		logger.Error("initial check failed", "error", err)
		// Continue running even if initial check fails
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := checkAndUpdate(ctx, cfg, apiClient, &lastIP, logger); err != nil {
				logger.Error("check failed", "error", err)
			}

		case sig := <-sigChan:
			logger.Info("received signal, shutting down", "signal", sig)
			return nil
		}
	}
}

// runCron runs a single check and exits.
func runCron(cfg Config, logger *slog.Logger) error {
	logger.Info("running one-shot update",
		"owner", cfg.Owner,
		"location", cfg.Location,
		"ipv6", cfg.IPv6,
	)

	// Initialize state manager
	stateMgr, err := state.NewManager(cfg.StateDir)
	if err != nil {
		return err
	}

	// Detect current IP
	version := pubip.IPv4
	if cfg.IPv6 {
		version = pubip.IPv6
	}

	currentIP, err := pubip.IP(version)
	if err != nil {
		return fmt.Errorf("failed to detect IP: %w", err)
	}

	logger.Debug("detected IP", "ip", currentIP)

	// Check if IP changed since last run
	changed, err := stateMgr.HasIPChanged(cfg.Owner, cfg.Location, currentIP)
	if err != nil {
		return err
	}

	if !changed {
		logger.Info("IP unchanged, skipping update")
		return nil
	}

	logger.Info("IP changed, updating DNS", "ip", currentIP)

	// Create API client and call API
	apiClient := client.New(client.Config{
		APIURL: cfg.APIURL,
		APIKey: cfg.APIKey,
	})

	ctx, ctxCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer ctxCancel()

	resp, err := apiClient.UpdateDNS(ctx, cfg.Owner, cfg.Location, currentIP)
	if err != nil {
		return err
	}

	// Save new state
	newState := &state.State{
		IPHash:    state.HashIP(currentIP),
		UpdatedAt: time.Now().UTC(),
	}
	if err := stateMgr.Save(cfg.Owner, cfg.Location, newState); err != nil {
		logger.Warn("failed to save state", "error", err)
		// Don't fail the operation if state save fails
	}

	if resp.Changed {
		logger.Info("DNS updated successfully",
			"subdomain", resp.Subdomain,
			"ip", resp.IP,
		)
	} else {
		logger.Info("DNS unchanged (server already had this IP)")
	}

	return nil
}

// checkAndUpdate detects the current IP and updates DNS if it changed.
// Used by daemon mode with in-memory state tracking.
func checkAndUpdate(ctx context.Context, cfg Config, apiClient *client.Client, lastIP *string, logger *slog.Logger) error {
	// Detect current IP
	version := pubip.IPv4
	if cfg.IPv6 {
		version = pubip.IPv6
	}

	currentIP, err := pubip.IP(version)
	if err != nil {
		return fmt.Errorf("failed to detect IP: %w", err)
	}

	logger.Debug("detected IP", "ip", currentIP)

	// Check if IP changed
	if currentIP == *lastIP {
		logger.Debug("IP unchanged, skipping update")
		return nil
	}

	logger.Info("IP changed, updating DNS",
		"old", *lastIP,
		"new", currentIP,
	)

	// Call API with client-detected IP
	resp, err := apiClient.UpdateDNS(ctx, cfg.Owner, cfg.Location, currentIP)
	if err != nil {
		return err
	}

	// Update last known IP
	*lastIP = currentIP

	if resp.Changed {
		logger.Info("DNS updated successfully",
			"subdomain", resp.Subdomain,
			"ip", resp.IP,
		)
	} else {
		logger.Debug("DNS unchanged (server already had this IP)")
	}

	return nil
}
