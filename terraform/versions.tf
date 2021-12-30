terraform {

  backend "s3" {
    bucket = "terraform-remote-state-vf8d"
    key    = "platform-cost-report"
    region = "eu-west-1"
    acl    = "bucket-owner-full-control"
  }

  required_version = "1.1.2"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3"
    }
  }
}
