resource "aws_lambda_function_url" "fn_url" {
  function_name = var.function_name
  authorization_type = "NONE"
  cors {
    allow_credentials = false
    allow_headers = ["*"]
    allow_methods = ["GET","POST","OPTIONS"]
    allow_origins = ["*"]
  }
}

output "function_url" {
  value = aws_lambda_function_url.fn_url.function_url
}
