variable "function_name" {
	type = string
}

variable "s3_bucket" {
	type = string
}

variable "s3_key" {
	type = string
}

variable "handler" {
	type    = string
	default = "main"
}

variable "runtime" {
	type    = string
	default = "go1.x"
}

variable "role_arn" {
	type = string
}

variable "environment" {
	type    = map(string)
	default = {}
}

variable "memory_size" {
	type    = number
	default = 128
}

variable "timeout" {
	type    = number
	default = 30
}

variable "publish" {
	type    = bool
	default = true
}
