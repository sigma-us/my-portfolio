terraform {
  backend "s3" {
    bucket   = "portfolio-tf"
    key      = "production/terraform.tfstate"
    region   = "us-east-1"
    acl      = "bucket-owner-full-control"
    profile  = "personal"
  }
}

provider "aws" {
  shared_credentials_file = "~/.aws/credentials"
  profile                 = var.aws_profile
  region                  = var.aws_region
}
