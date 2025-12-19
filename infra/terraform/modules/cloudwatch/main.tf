resource "aws_cloudwatch_metric_alarm" "dlq_alarm" {
  alarm_name          = "${var.dlq_queue_name}-dlq-alarm"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = 1
  metric_name         = "ApproximateNumberOfMessagesVisible"
  namespace           = "AWS/SQS"
  statistic           = "Average"
  period              = 60
  threshold           = var.thresholds["dlq_messages"]

  dimensions = {
    QueueName = var.dlq_queue_name
  }
}
