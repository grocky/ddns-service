variable "app_environment" {
  default = "test"
}

resource "aws_api_gateway_rest_api" "ddns-service" {
  name        = "ddns-service"
  description = "Dynamic DNS service interface"
}

resource "aws_api_gateway_resource" "proxy" {
  rest_api_id = "${aws_api_gateway_rest_api.ddns-service.id}"
  parent_id   = "${aws_api_gateway_rest_api.ddns-service.root_resource_id}"
  path_part   = "{proxy+}"
}

resource "aws_api_gateway_method" "proxy" {
  rest_api_id   = "${aws_api_gateway_rest_api.ddns-service.id}"
  resource_id   = "${aws_api_gateway_resource.proxy.id}"
  http_method   = "ANY"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "lambda" {
  rest_api_id = "${aws_api_gateway_rest_api.ddns-service.id}"
  resource_id = "${aws_api_gateway_method.proxy.resource_id}"
  http_method = "${aws_api_gateway_method.proxy.http_method}"

  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = "${aws_lambda_function.ddns-service.invoke_arn}"
}

resource "aws_api_gateway_method" "proxy_root" {
  rest_api_id   = "${aws_api_gateway_rest_api.ddns-service.id}"
  resource_id   = "${aws_api_gateway_rest_api.ddns-service.root_resource_id}"
  http_method   = "ANY"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "lambda_root" {
  rest_api_id = "${aws_api_gateway_rest_api.ddns-service.id}"
  resource_id = "${aws_api_gateway_method.proxy_root.resource_id}"
  http_method = "${aws_api_gateway_method.proxy_root.http_method}"

  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = "${aws_lambda_function.ddns-service.invoke_arn}"
}

resource "aws_api_gateway_deployment" "ddns-service" {
  depends_on = [
    "aws_api_gateway_integration.lambda",
    "aws_api_gateway_integration.lambda_root",
  ]

  rest_api_id = "${aws_api_gateway_rest_api.ddns-service.id}"
  stage_name  = "${var.app_environment}"
}

resource "aws_api_gateway_base_path_mapping" "base-path" {
  api_id      = "${aws_api_gateway_rest_api.ddns-service.id}"
  stage_name  = "${aws_api_gateway_deployment.ddns-service.stage_name}"
  domain_name = "${aws_api_gateway_domain_name.ddns-service.domain_name}"
}

output "gateway_base_url" {
  value = "${aws_api_gateway_deployment.ddns-service.invoke_url}"
}
