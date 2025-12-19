variable "lambda_name" {
  type = string
}

variable "dlq_queue_name" {
  type = string
}

variable "thresholds" {
  type = map(number)
  default = {
    dlq_messages  = 1
    lambda_errors = 1
  }
}
