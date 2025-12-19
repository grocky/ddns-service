resource "aws_route53_zone" "ddns" {
  name = local.domain_name

  tags = {
    Name        = "ddns-zone"
    Environment = var.environment
    Application = "ddns-service"
  }
}

output "ddns_hosted_zone_id" {
  description = "The hosted zone ID for the DDNS domain"
  value       = aws_route53_zone.ddns.zone_id
}

output "ddns_nameservers" {
  description = "The nameservers"
  value       = aws_route53_zone.ddns.name_servers
}
