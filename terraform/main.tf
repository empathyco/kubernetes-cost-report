provider "aws" {

  region = "eu-west-1"

  assume_role {
    role_arn = var.terraform_role_ARN
  }

  default_tags {
    tags = {
      BusinessUnit = "EmpathyPlatform"
      Department   = "Platform"
      Owner        = "platform@empathy.co"
      Workload     = "Cost-report"
      Management   = "Terraform"
      Source       = "github.com/empathyco/platform-cost-report"
      Environment  = var.environment_tag
    }
  }
}
