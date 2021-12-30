variable "terraform_role_ARN" {
  type        = string
  description = "ARN of IAM role"
}

variable "role_name" {
  type        = string
  description = "IAM role name"
}

variable "oidc_url" {
  type        = string
  description = "OIDC url"
}
