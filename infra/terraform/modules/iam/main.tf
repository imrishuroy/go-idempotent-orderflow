resource "aws_iam_role" "lambda_exec" {
  name = "${var.lambda_name}-exec-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Action = "sts:AssumeRole",
        Principal = {
          Service = "lambda.amazonaws.com"
        },
        Effect = "Allow",
        Sid = ""
      }
    ]
  })
}

data "aws_iam_policy_document" "lambda_policy_doc" {
  statement {
    sid     = "DynamoDBAccess"
    effect  = "Allow"
    actions = [
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:UpdateItem",
      "dynamodb:TransactWriteItems"
    ]
    resources = var.dynamodb_table_arns
  }

  statement {
    sid     = "SQSAccess"
    effect  = "Allow"
    actions = [
      "sqs:SendMessage",
      "sqs:ReceiveMessage",
      "sqs:DeleteMessage",
      "sqs:GetQueueAttributes",
      "sqs:GetQueueUrl"
    ]
    resources = [var.sqs_queue_arn]
  }

  statement {
    sid     = "CloudWatchLogs"
    effect  = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = ["arn:aws:logs:*:*:*"]
  }

  statement {
    sid    = "CloudWatchMetrics"
    effect = "Allow"
    actions = [
      "cloudwatch:PutMetricData"
    ]
    resources = ["*"]
  }
}

resource "aws_iam_policy" "lambda_policy" {
  name   = "${var.lambda_name}-policy"
  policy = data.aws_iam_policy_document.lambda_policy_doc.json
}

resource "aws_iam_role_policy_attachment" "attach" {
  role       = aws_iam_role.lambda_exec.name
  policy_arn = aws_iam_policy.lambda_policy.arn
}
