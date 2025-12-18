# =============================================================================
# DynamoDB Table
# =============================================================================

resource "aws_dynamodb_table" "ip_mappings" {
  name         = "DdnsServiceIpMapping"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "OwnerId"
  range_key    = "LocationName"

  attribute {
    name = "OwnerId"
    type = "S"
  }

  attribute {
    name = "LocationName"
    type = "S"
  }

  ttl {
    attribute_name = "TimeToExist"
    enabled        = false
  }

  tags = {
    Name        = "DdnsServiceIpMapping"
    Environment = var.environment
    Application = "ddns-service"
  }
}
