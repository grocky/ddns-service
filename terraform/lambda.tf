variable "app_version" { }

data "template_file" "ddns-service-lambda-assume-role-policy" {
  template = "${file("${path.module}/lambda-assume-role-policy.json")}"
}

resource "aws_iam_role" "lambda_exec" {
  name = "lambda_exec"
  assume_role_policy = "${data.template_file.ddns-service-lambda-assume-role-policy.rendered}"
}

resource "aws_lambda_function" "ddns-service" {
  function_name = "ddns-service"

  s3_bucket = "grocky-services"
  s3_key    = "ddns-service-${var.app_version}.zip"

  handler = "ddns-service_linux_${var.app_version}"
  runtime = "go1.x"

  role = "${aws_iam_role.lambda_exec.arn}"
}

data "template_file" "ddns-service-lambda-role-policy" {
  template = "${file("${path.module}/lambda-role-policy.json")}"
}

resource "aws_iam_policy" "ddns-service-policy" {
  name   = "ddns-service-policy"
  path   = "/"
  policy = "${data.template_file.ddns-service-lambda-role-policy.rendered}"
}

resource "aws_iam_policy_attachment" "ddns-service_policy_attachment" {
  name       = "ddns-service-policy-attachment"
  roles      = ["${aws_iam_role.lambda_exec.name}"]
  policy_arn = "${aws_iam_policy.ddns-service-policy.arn}"
}

resource "aws_lambda_permission" "apigw" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = "${aws_lambda_function.ddns-service.arn}"
  principal     = "apigateway.amazonaws.com"

  # The /*/* portion grants access from any method on any resource
  # within the API Gateway "REST API".
  source_arn = "${aws_api_gateway_deployment.ddns-service.execution_arn}/*/*"
}

