variable "lambda_name" {
	type = string
}

variable "dynamodb_table_arns" {
	type = list(string)
}

variable "sqs_queue_arn" {
	type = string
}

variable "cloudwatch_namespace" {
	type    = string
	default = "orders-app"
}
