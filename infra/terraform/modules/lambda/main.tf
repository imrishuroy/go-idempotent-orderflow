resource "aws_lambda_function" "this" {
  function_name = var.function_name
  s3_bucket     = var.s3_bucket
  s3_key        = var.s3_key
  handler       = var.handler
  runtime       = var.runtime
  role          = var.role_arn
  memory_size   = var.memory_size
  timeout       = var.timeout
  publish       = var.publish

  environment {
    variables = var.environment
  }

  tracing_config {
    mode = "Active"
  }

  tags = {
    Name = var.function_name
  }
}
