# =============================================================================
# IAM Role for Lambda
# =============================================================================

resource "aws_iam_role" "lambda" {
  name = "ddns-service-lambda-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "ddns-service-lambda-${var.environment}"
    Environment = var.environment
    Application = "ddns-service"
  }
}

# Basic Lambda execution policy (CloudWatch Logs)
resource "aws_iam_role_policy_attachment" "lambda_basic" {
  role       = aws_iam_role.lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# DynamoDB access policy
resource "aws_iam_role_policy" "dynamodb" {
  name = "dynamodb-access"
  role = aws_iam_role.lambda.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:UpdateItem",
          "dynamodb:DeleteItem",
          "dynamodb:Query",
          "dynamodb:Scan"
        ]
        Resource = [
          aws_dynamodb_table.ip_mappings.arn,
          aws_dynamodb_table.owners.arn
        ]
      }
    ]
  })
}

# SES email sending policy
resource "aws_iam_role_policy" "ses" {
  name = "ses-send-email"
  role = aws_iam_role.lambda.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ses:SendEmail",
          "ses:SendRawEmail"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "ses:FromAddress" = "noreply@rockygray.com"
          }
        }
      }
    ]
  })
}

# Route53 DNS record management policy
resource "aws_iam_role_policy" "route53" {
  name = "route53-dns-management"
  role = aws_iam_role.lambda.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "route53:ChangeResourceRecordSets",
          "route53:ListResourceRecordSets"
        ]
        Resource = aws_route53_zone.ddns.arn
      },
      {
        Effect = "Allow"
        Action = [
          "route53:GetHostedZone"
        ]
        Resource = aws_route53_zone.ddns.arn
      }
    ]
  })
}

# =============================================================================
# CloudWatch Log Group
# =============================================================================

resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/ddns-service"
  retention_in_days = 14

  tags = {
    Name        = "ddns-service-logs"
    Environment = var.environment
    Application = "ddns-service"
  }
}

# =============================================================================
# Lambda Function
# =============================================================================

resource "aws_lambda_function" "ddns_service" {
  function_name = "ddns-service"
  role          = aws_iam_role.lambda.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]

  filename         = "${path.module}/../scripts/dist/ddns-service.zip"
  source_code_hash = filebase64sha256("${path.module}/../scripts/dist/ddns-service.zip")

  memory_size = 128
  timeout     = 10

  environment {
    variables = {
      ENVIRONMENT            = var.environment
      ROUTE53_HOSTED_ZONE_ID = aws_route53_zone.ddns.zone_id
    }
  }

  depends_on = [
    aws_cloudwatch_log_group.lambda,
    aws_iam_role_policy_attachment.lambda_basic
  ]

  tags = {
    Name        = "ddns-service-${var.environment}"
    Environment = var.environment
    Application = "ddns-service"
  }
}

# =============================================================================
# API Gateway Permission
# =============================================================================

resource "aws_lambda_permission" "apigw" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.ddns_service.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.ddns_service.execution_arn}/*/*"
}
