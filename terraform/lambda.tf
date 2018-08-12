provider "aws" {
  region = "us-east-1"
}

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
  s3_key    = "ddns-service-1.0.0.zip"

  handler = "index.handler"
  runtime = "nodejs8.10"

  role = "${aws_iam_role.lambda_exec.arn}"
}

