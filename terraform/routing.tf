# =============================================================================
# Route53 Zone
# =============================================================================

data "aws_route53_zone" "main" {
  zone_id      = "Z33Z8O8Z1ZA8HH"
  private_zone = false
}

# =============================================================================
# API Gateway Custom Domain
# =============================================================================

resource "aws_api_gateway_domain_name" "ddns_service" {
  domain_name              = "ddns.rockygray.com"
  regional_certificate_arn = aws_acm_certificate_validation.cert.certificate_arn

  endpoint_configuration {
    types = ["REGIONAL"]
  }

  tags = {
    Name        = "ddns-service-domain"
    Environment = var.environment
    Application = "ddns-service"
  }
}

# =============================================================================
# DNS Record
# =============================================================================

resource "aws_route53_record" "ddns" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = aws_api_gateway_domain_name.ddns_service.domain_name
  type    = "A"

  alias {
    name                   = aws_api_gateway_domain_name.ddns_service.regional_domain_name
    zone_id                = aws_api_gateway_domain_name.ddns_service.regional_zone_id
    evaluate_target_health = true
  }
}

# =============================================================================
# Outputs
# =============================================================================

output "custom_domain" {
  description = "Custom domain endpoint"
  value       = "https://${aws_api_gateway_domain_name.ddns_service.domain_name}"
}
