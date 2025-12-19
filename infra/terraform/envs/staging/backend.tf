terraform {
  backend "s3" {
    bucket        = "orderflow-tf-state"
    key           = "staging/terraform.tfstate"
    region        = "ap-south-1"
    use_lockfile  = true
  }
}
