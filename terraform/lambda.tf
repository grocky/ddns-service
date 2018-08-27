provider "aws" {
  region = "us-east-1"
}

variable "app_version" { }

resource "aws_iam_role" "lambda_exec" {
  name = "lambda_exec"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_lambda_function" "ddns-service" {
  function_name = "ddns-service"

  s3_bucket = "grocky-services"
  s3_key    = "ddns-service-${var.app_version}.zip"

  handler = "ddns-service_linux_${var.app_version}"
  runtime = "go1.x"

  role = "${aws_iam_role.lambda_exec.arn}"
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

