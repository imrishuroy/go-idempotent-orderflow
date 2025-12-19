locals {
  queue_name = var.enable_fifo ? "${var.name_prefix}.fifo" : "${var.name_prefix}"
  dlq_name   = var.enable_fifo ? "${var.name_prefix}-dlq.fifo" : "${var.name_prefix}-dlq"
}

resource "aws_sqs_queue" "dlq" {
  name                       = local.dlq_name
  visibility_timeout_seconds = var.visibility_timeout_seconds
}

resource "aws_sqs_queue" "main" {
  name                       = local.queue_name
  visibility_timeout_seconds = var.visibility_timeout_seconds

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dlq.arn
    maxReceiveCount     = var.max_receive_count
  })

  fifo_queue = var.enable_fifo ? true : false
}
