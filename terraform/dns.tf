resource "aws_route53_zone" "ddns" {
  name = var.domain_name

  tags = {
    Name        = "ddns-zone"
    Environment = var.environment
    Application = "ddns-service"
  }
}

output "ddns_hosted_zone_id" {
  description = "The hosted zone ID for the DDNS domain (${var.domain_name})"
  value       = aws_route53_zone.ddns.zone_id
}

output "ddns_nameservers" {
  description = "The nameservers for ${var.domain_name}"
  value       = aws_route53_zone.ddns.name_servers
}
