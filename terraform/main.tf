terraform {
  backend "s3" {
    encrypt = true
    bucket = "grocky-tfstate"
    dynamodb_table = "tfstate-lock"
    region = "us-east-1"
    key = "ddns.rockygray.com/terraform.tfstate"
  }
}

provider "aws" {
  region = "us-east-1"
}
