resource "aws_api_gateway_rest_api" "ddns-service" {
  name        = "ddns-service"
  description = "Dynamic DNS service interface"
}

