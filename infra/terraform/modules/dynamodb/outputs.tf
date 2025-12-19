output "orders_table_name" {
  value = aws_dynamodb_table.orders.name
}
output "idempotency_table_name" {
  value = aws_dynamodb_table.idempotency.name
}
output "orders_table_arn" {
  value = aws_dynamodb_table.orders.arn
}
output "idempotency_table_arn" {
  value = aws_dynamodb_table.idempotency.arn
}
