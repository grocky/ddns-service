data "aws_route53_zone" "selected" {
  zone_id = "Z33Z8O8Z1ZA8HH"
  private_zone = false
}

resource "aws_api_gateway_domain_name" "ddns-service" {
  domain_name = "ddns.rockygray.com"
  endpoint_configuration {
    types = ["REGIONAL"]
  }
  regional_certificate_arn = "${aws_acm_certificate_validation.cert.certificate_arn}"
}

resource "aws_route53_record" "ddns" {
  zone_id = "${data.aws_route53_zone.selected.id}"
  name = "${aws_api_gateway_domain_name.ddns-service.domain_name}"
  type = "A"
  alias {
    name = "${aws_api_gateway_domain_name.ddns-service.regional_domain_name}"
    zone_id = "${aws_api_gateway_domain_name.ddns-service.regional_zone_id}"
    evaluate_target_health = true
  }
}

output "custom_domain_distribution" {
  value = "${aws_api_gateway_domain_name.ddns-service.regional_domain_name}"
}
