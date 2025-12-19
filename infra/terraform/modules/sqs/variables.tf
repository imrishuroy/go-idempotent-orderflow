variable "name_prefix" {
	type = string
}

variable "max_receive_count" {
	type    = number
	default = 5
}

variable "visibility_timeout_seconds" {
	type    = number
	default = 60
}

variable "enable_fifo" {
	type    = bool
	default = false
}
