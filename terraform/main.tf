terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket       = "grocky-tfstate"
    key          = "ddns.grocky.net/terraform.tfstate"
    region       = "us-east-1"
    use_lockfile = true
    encrypt      = true
  }
}

provider "aws" {
  region = "us-east-1"
}

# =============================================================================
# Variables
# =============================================================================

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "prod"
}

locals {
  domain_name = "grocky.net"
}

# =============================================================================
# Data Sources
# =============================================================================

data "aws_caller_identity" "current" {}
data "aws_region" "current" {}
