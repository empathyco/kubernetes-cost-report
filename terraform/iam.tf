module "cost_report_iam_eks_role" {
  source      = "terraform-aws-modules/iam/aws//modules/iam-eks-role"
  version     = "4.10.1"
  create_role = true
  role_name   = var.role_name

  cluster_service_accounts = {
    "${var.cluster_name}" = ["${local.namespace}:${local.serviceaccount}"]
  }

  role_policy_arns = [aws_iam_policy.cost_report_policy.arn]
}

data "aws_iam_policy_document" "policy" {
  statement {
    sid       = ""
    effect    = "Allow"
    resources = ["*"] #tfsec:ignore:aws-iam-no-policy-wildcards
    actions   = ["pricing:*"]
  }

  statement {
    sid       = ""
    effect    = "Allow"
    resources = ["*"] #tfsec:ignore:aws-iam-no-policy-wildcards
    actions   = ["ec2:Describe*"]
  }
}
resource "aws_iam_policy" "cost_report_policy" {
  name        = "cost_report_policy"
  description = "cost_report_policy"
  policy      = data.aws_iam_policy_document.policy.json
}
