package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/grocky/ddns-service/internal/admin"
)

const (
	// Default values - can be overridden with flags
	defaultTableName    = "DdnsServiceIpMapping"
	defaultHostedZoneID = "Z030530123PNW3FUSMWUZ"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "change-subdomain":
		changeSubdomainCmd(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`ddns-admin - Administrative tool for DDNS Service

Usage:
  ddns-admin <command> [options]

Commands:
  change-subdomain  Change the subdomain for an owner's location
  help              Show this help message

Examples:
  ddns-admin change-subdomain --owner grocky --location home --subdomain home

Run 'ddns-admin <command> --help' for more information on a command.`)
}

func changeSubdomainCmd(args []string) {
	fs := flag.NewFlagSet("change-subdomain", flag.ExitOnError)

	owner := fs.String("owner", "", "Owner ID (required)")
	location := fs.String("location", "", "Location name (required)")
	subdomain := fs.String("subdomain", "", "New subdomain (required, without domain suffix)")
	tableName := fs.String("table", defaultTableName, "DynamoDB table name")
	hostedZoneID := fs.String("zone-id", defaultHostedZoneID, "Route53 hosted zone ID")
	dryRun := fs.Bool("dry-run", false, "Show what would be changed without making changes")
	verbose := fs.Bool("verbose", false, "Enable verbose logging")

	fs.Usage = func() {
		fmt.Println(`Change the subdomain for an owner's location.

This command updates both Route53 and DynamoDB to use a custom subdomain
instead of the auto-generated hash-based subdomain.

Usage:
  ddns-admin change-subdomain [options]

Options:`)
		fs.PrintDefaults()
		fmt.Println(`
Examples:
  # Change subdomain for grocky/home to home.grocky.net
  ddns-admin change-subdomain --owner grocky --location home --subdomain home

  # Dry run to see what would change
  ddns-admin change-subdomain --owner grocky --location home --subdomain home --dry-run`)
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Validate required flags
	if *owner == "" || *location == "" || *subdomain == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner, --location, and --subdomain are required")
		fs.Usage()
		os.Exit(1)
	}

	// Set up logger
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	// Load AWS config
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("failed to load AWS config", "error", err)
		os.Exit(1)
	}

	// Create clients
	dynamoClient := dynamodb.NewFromConfig(cfg)
	route53Client := route53.NewFromConfig(cfg)

	// Create service
	svc := admin.NewSubdomainService(dynamoClient, route53Client, *tableName, *hostedZoneID, logger)

	input := admin.ChangeSubdomainInput{
		OwnerID:      *owner,
		Location:     *location,
		NewSubdomain: *subdomain,
	}

	if *dryRun {
		fmt.Println("Dry run mode - no changes will be made")
		fmt.Println()
		fmt.Printf("Would change subdomain for:\n")
		fmt.Printf("  Owner:    %s\n", input.OwnerID)
		fmt.Printf("  Location: %s\n", input.Location)
		fmt.Printf("  New subdomain: %s.grocky.net\n", input.NewSubdomain)
		return
	}

	// Execute the change
	result, err := svc.ChangeSubdomain(ctx, input)
	if err != nil {
		logger.Error("failed to change subdomain", "error", err)
		os.Exit(1)
	}

	fmt.Println("Subdomain changed successfully!")
	fmt.Println()
	fmt.Printf("  Old: %s\n", result.OldFQDN)
	fmt.Printf("  New: %s\n", result.NewFQDN)
	fmt.Printf("  IP:  %s\n", result.IP)
	fmt.Println()
	fmt.Println("DNS propagation may take a few minutes.")
}
