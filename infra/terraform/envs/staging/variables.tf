variable "project_name" { default = "order-processor" }
variable "aws_region" { default = "ap-south-1" }
variable "lambda_s3_bucket" { type = string }
variable "lambda_api_s3_key" { type = string }
variable "lambda_worker_s3_key" { type = string }
