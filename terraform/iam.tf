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
resource "aws_iam_policy" "cost_report_policy" {
  name = "cost_report_policy"
  description = "cost_report_policy"
  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": [
                "pricing:*"
            ],
            "Effect": "Allow",
            "Resource": "*"
        },
        {
            "Action": [
                "ec2:Describe*"
            ],
            "Effect": "Allow",
            "Resource": "*"
        }
    ]
}
EOF
}
