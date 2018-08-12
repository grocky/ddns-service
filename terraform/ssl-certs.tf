resource "aws_acm_certificate" "wildcard-cert" {
  domain_name = "ddns.rockygray.com"
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "cert_validation" {
  name = "${aws_acm_certificate.wildcard-cert.domain_validation_options.0.resource_record_name}"
  type = "${aws_acm_certificate.wildcard-cert.domain_validation_options.0.resource_record_type}"
  zone_id = "${data.aws_route53_zone.selected.id}"
  records = [
    "${aws_acm_certificate.wildcard-cert.domain_validation_options.0.resource_record_value}"
  ]
  ttl = 60
}

resource "aws_acm_certificate_validation" "cert" {
  certificate_arn = "${aws_acm_certificate.wildcard-cert.arn}"
  validation_record_fqdns = ["${aws_route53_record.cert_validation.fqdn}"]
}

