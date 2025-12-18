# =============================================================================
# API Gateway REST API
# =============================================================================

resource "aws_api_gateway_rest_api" "ddns_service" {
  name        = "ddns-service"
  description = "Dynamic DNS service API"

  endpoint_configuration {
    types = ["REGIONAL"]
  }

  tags = {
    Name        = "ddns-service-api"
    Environment = var.environment
    Application = "ddns-service"
  }
}

# =============================================================================
# Proxy Resource (catches all paths)
# =============================================================================

resource "aws_api_gateway_resource" "proxy" {
  rest_api_id = aws_api_gateway_rest_api.ddns_service.id
  parent_id   = aws_api_gateway_rest_api.ddns_service.root_resource_id
  path_part   = "{proxy+}"
}

resource "aws_api_gateway_method" "proxy" {
  rest_api_id   = aws_api_gateway_rest_api.ddns_service.id
  resource_id   = aws_api_gateway_resource.proxy.id
  http_method   = "ANY"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "proxy" {
  rest_api_id             = aws_api_gateway_rest_api.ddns_service.id
  resource_id             = aws_api_gateway_resource.proxy.id
  http_method             = aws_api_gateway_method.proxy.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.ddns_service.invoke_arn
}

# =============================================================================
# Root Resource
# =============================================================================

resource "aws_api_gateway_method" "root" {
  rest_api_id   = aws_api_gateway_rest_api.ddns_service.id
  resource_id   = aws_api_gateway_rest_api.ddns_service.root_resource_id
  http_method   = "ANY"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "root" {
  rest_api_id             = aws_api_gateway_rest_api.ddns_service.id
  resource_id             = aws_api_gateway_rest_api.ddns_service.root_resource_id
  http_method             = aws_api_gateway_method.root.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.ddns_service.invoke_arn
}

# =============================================================================
# Deployment & Stage
# =============================================================================

resource "aws_api_gateway_deployment" "ddns_service" {
  rest_api_id = aws_api_gateway_rest_api.ddns_service.id

  triggers = {
    redeployment = sha1(jsonencode([
      aws_api_gateway_resource.proxy.id,
      aws_api_gateway_method.proxy.id,
      aws_api_gateway_integration.proxy.id,
      aws_api_gateway_method.root.id,
      aws_api_gateway_integration.root.id,
    ]))
  }

  lifecycle {
    create_before_destroy = true
  }

  depends_on = [
    aws_api_gateway_integration.proxy,
    aws_api_gateway_integration.root,
  ]
}

resource "aws_api_gateway_stage" "ddns_service" {
  deployment_id = aws_api_gateway_deployment.ddns_service.id
  rest_api_id   = aws_api_gateway_rest_api.ddns_service.id
  stage_name    = var.environment

  tags = {
    Name        = "ddns-service-stage-${var.environment}"
    Environment = var.environment
    Application = "ddns-service"
  }
}

# =============================================================================
# Custom Domain Mapping
# =============================================================================

resource "aws_api_gateway_base_path_mapping" "ddns_service" {
  api_id      = aws_api_gateway_rest_api.ddns_service.id
  stage_name  = aws_api_gateway_stage.ddns_service.stage_name
  domain_name = aws_api_gateway_domain_name.ddns_service.domain_name
}

# =============================================================================
# Outputs
# =============================================================================

output "api_gateway_url" {
  description = "API Gateway invoke URL"
  value       = aws_api_gateway_stage.ddns_service.invoke_url
}
