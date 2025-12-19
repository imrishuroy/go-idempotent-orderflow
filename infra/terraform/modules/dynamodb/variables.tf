variable "name_prefix" {
  type        = string
  description = "Prefix for DynamoDB table names (e.g., project-env)"
}

variable "billing_mode" {
  type    = string
  default = "PAY_PER_REQUEST"
}

variable "orders_table_name" {
  type        = string
  description = "Orders table name (optional override)"
  default     = ""
}

variable "idempotency_table_name" {
  type        = string
  description = "Idempotency table name (optional override)"
  default     = ""
}
