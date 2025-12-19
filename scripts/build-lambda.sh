#!/bin/bash
set -e

# Build script for the DDNS Service Lambda function
# Produces a deployment package for AWS Lambda (ARM64, provided.al2023)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
DIST_DIR="$ROOT_DIR/dist"
LAMBDA_ZIP_NAME="ddns-service.zip"

echo "Building ddns-service-lambda..."

# Create dist directory
mkdir -p "$DIST_DIR"

# Build the Go binary for Lambda (ARM64, Linux)
cd "$ROOT_DIR"
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
    -tags lambda.norpc \
    -ldflags="-s -w" \
    -o "$DIST_DIR/bootstrap" \
    ./cmd/ddns-service-lambda

# Create the deployment zip
cd "$DIST_DIR"
rm -f "$LAMBDA_ZIP_NAME"
zip "$LAMBDA_ZIP_NAME" bootstrap

echo "Build complete: $DIST_DIR/$LAMBDA_ZIP_NAME"
ls -lh "$DIST_DIR/$LAMBDA_ZIP_NAME"
