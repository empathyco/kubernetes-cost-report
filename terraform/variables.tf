variable "terraform_role_ARN" {
  type        = string
  description = "ARN of IAM role"
}

variable "role_name" {
  type        = string
  description = "IAM role name"
}

variable "cluster_name" {
  type        = string
  description = "Cluster name"
}

variable "environment_tag" {
  type        = string
  description = "Tag for the environment"
}
