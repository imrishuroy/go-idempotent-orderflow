provider "aws" {
  region = var.aws_region
}

module "dynamodb" {
  source = "../../modules/dynamodb"
  name_prefix = "${var.project_name}-staging"
  billing_mode = "PAY_PER_REQUEST"
}

module "sqs" {
  source = "../../modules/sqs"
  name_prefix = "${var.project_name}-orders-queue-staging"
  max_receive_count = 5
  visibility_timeout_seconds = 60
}

module "iam_api" {
  source = "../../modules/iam"
  lambda_name = "${var.project_name}-api-staging"
  dynamodb_table_arns = [module.dynamodb.orders_table_arn, module.dynamodb.idempotency_table_arn]
  sqs_queue_arn = module.sqs.queue_arn
}

module "lambda_api" {
  source = "../../modules/lambda"
  function_name = "${var.project_name}-api-staging"
  s3_bucket = var.lambda_s3_bucket
  s3_key = var.lambda_api_s3_key
  role_arn = module.iam_api.lambda_role_arn
  environment = {
    IDEMPOTENCY_TABLE = module.dynamodb.idempotency_table_name
    ORDERS_TABLE = module.dynamodb.orders_table_name
    ORDERS_QUEUE_URL = module.sqs.queue_url
  }
}

module "iam_worker" {
  source = "../../modules/iam"
  lambda_name = "${var.project_name}-worker-staging"
  dynamodb_table_arns = [module.dynamodb.orders_table_arn, module.dynamodb.idempotency_table_arn]
  sqs_queue_arn = module.sqs.queue_arn
}

module "lambda_worker" {
  source = "../../modules/lambda"
  function_name = "${var.project_name}-worker-staging"
  s3_bucket = var.lambda_s3_bucket
  s3_key = var.lambda_worker_s3_key
  role_arn = module.iam_worker.lambda_role_arn
  environment = {
    IDEMPOTENCY_TABLE = module.dynamodb.idempotency_table_name
    ORDERS_TABLE = module.dynamodb.orders_table_name
  }
}

# Event source mapping (SQS -> Lambda)
resource "aws_lambda_event_source_mapping" "sqs_to_worker" {
  event_source_arn = module.sqs.queue_arn
  function_name    = module.lambda_worker.lambda_arn
  batch_size       = 1
  enabled          = true
}

output "api_function_url" {
  value = module.lambda_api.function_name # if using Function URL, use its output
}
