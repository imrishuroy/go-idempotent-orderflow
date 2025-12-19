locals {
  orders_table_name      = length(var.orders_table_name) > 0 ? var.orders_table_name : "${var.name_prefix}-orders"
  idempotency_table_name = length(var.idempotency_table_name) > 0 ? var.idempotency_table_name : "${var.name_prefix}-idempotency"
}

resource "aws_dynamodb_table" "orders" {
  name         = local.orders_table_name
  billing_mode = var.billing_mode
  hash_key     = "order_id"

  attribute {
    name = "order_id"
    type = "S"
  }

  # example GSI for customer_id -> created_at
  attribute {
    name = "customer_id"
    type = "S"
  }
  attribute {
    name = "created_at"
    type = "S"
  }

  global_secondary_index {
    name            = "customer_id_index"
    hash_key        = "customer_id"
    range_key       = "created_at"
    projection_type = "ALL"
  }

  tags = {
    Name = local.orders_table_name
  }
}

resource "aws_dynamodb_table" "idempotency" {
  name         = local.idempotency_table_name
  billing_mode = var.billing_mode
  hash_key     = "idempotency_key"

  attribute {
    name = "idempotency_key"
    type = "S"
  }

  ttl {
    attribute_name = "expires_at"
    enabled        = true
  }

  tags = {
    Name = local.idempotency_table_name
  }
}
