resource "aws_dynamodb_table" "ddns-service-table" {
  name           = "DdnsServiceIpMapping"
  read_capacity  = 20
  write_capacity = 20
  hash_key       = "OwnerId"
  range_key      = "LocationName"

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
    enabled = false
  }

  tags {
    Name        = "DdnsServiceIpMapping"
    Environment = "production"
  }
}
