# =============================================================================
# SES Email Identity
# =============================================================================

# Note: The domain rockygray.com should already be verified in SES.
# If not, you'll need to verify it manually or add domain verification here.
# This creates an email identity for the sender address.

resource "aws_ses_email_identity" "noreply" {
  email = "noreply@rockygray.com"
}

# =============================================================================
# Output
# =============================================================================

output "ses_sender_email" {
  description = "SES verified sender email"
  value       = aws_ses_email_identity.noreply.email
}
