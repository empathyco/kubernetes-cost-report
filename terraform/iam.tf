module "cost_report_iam_assumable_role_with_oidc" {
  source                        = "terraform-aws-modules/iam/aws//modules/iam-assumable-role-with-oidc"
  version                       = "4.7.0"
  create_role                   = true
  role_name                     = var.role_name
  number_of_role_policy_arns    = 1
  provider_url                  = var.oidc_url
  role_policy_arns              = [aws_iam_policy.cost_report_policy.arn]
  oidc_fully_qualified_subjects = ["system:serviceaccount:${local.oidc.namespace}:${local.oidc.serviceaccount}"]
}

data "aws_iam_policy_document" "policy" {
  statement {
    sid       = ""
    effect    = "Allow"
    resources = ["*"]
    actions   = ["pricing:*"]
  }

  statement {
    sid       = ""
    effect    = "Allow"
    resources = ["*"]
    actions   = ["ec2:Describe*"]
  }
}
resource "aws_iam_policy" "cost_report_policy" {
  name = "cost_report_policy"
  description = "cost_report_policy"
# Test
  policy = data.aws_iam_policy_document.policy.json
}
