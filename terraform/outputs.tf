output "role" {
  value       = module.cost_report_iam_eks_role.iam_role_arn
  description = "The IAM role that can be assumed by the cost report lambda"
}
