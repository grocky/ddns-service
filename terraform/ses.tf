# =============================================================================
# SES Domain Identity
# =============================================================================

# Verify the domain for sending emails
resource "aws_ses_domain_identity" "main" {
  domain = local.domain_name
}

# DKIM for email authentication
resource "aws_ses_domain_dkim" "main" {
  domain = aws_ses_domain_identity.main.domain
}

# =============================================================================
# DNS Records for SES
# =============================================================================

# Domain verification TXT record
resource "aws_route53_record" "ses_verification" {
  zone_id = aws_route53_zone.ddns.zone_id
  name    = "_amazonses.${local.domain_name}"
  type    = "TXT"
  ttl     = 600
  records = [aws_ses_domain_identity.main.verification_token]
}

# DKIM CNAME records
resource "aws_route53_record" "ses_dkim" {
  count   = 3
  zone_id = aws_route53_zone.ddns.zone_id
  name    = "${aws_ses_domain_dkim.main.dkim_tokens[count.index]}._domainkey.${local.domain_name}"
  type    = "CNAME"
  ttl     = 600
  records = ["${aws_ses_domain_dkim.main.dkim_tokens[count.index]}.dkim.amazonses.com"]
}

# =============================================================================
# SES Email Identity
# =============================================================================

resource "aws_ses_email_identity" "noreply" {
  email = "noreply@${local.domain_name}"
}

# =============================================================================
# Output
# =============================================================================

output "ses_sender_email" {
  description = "SES verified sender email"
  value       = aws_ses_email_identity.noreply.email
}
