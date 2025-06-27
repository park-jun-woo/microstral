# https://parkjunwoo.com/microstral/deployments/terraform/kms.tf

data "aws_caller_identity" "current" {}

# KMS Key를 사용할 주체로 허용할 IAM Role의 Assume Policy 문서
data "aws_iam_policy_document" "kms_key_user_assume" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

# Role 생성
resource "aws_iam_role" "kms_key_user" {
  name               = "${var.project}-kms-key-user"
  assume_role_policy = data.aws_iam_policy_document.kms_key_user_assume.json
  tags               = local.tags
}

module "kms_cookie" {
  source  = "terraform-aws-modules/kms/aws"
  version = ">= 1.3.0"

  description              = "CloudFront 서명용 KMS 키"
  enable_key_rotation      = false
  deletion_window_in_days  = 7
  key_usage                = "SIGN_VERIFY"
  customer_master_key_spec = "RSA_2048"
  tags = local.tags

  # 키 소유자/관리자
  key_owners = [
    "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root",
    data.aws_caller_identity.current.arn,
  ]
  key_administrators = [
    "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root",
    data.aws_caller_identity.current.arn,
  ]
  key_users = [
    aws_iam_role.kms_key_user.arn,
  ]
}


data "aws_kms_public_key" "cookie" {
  key_id = module.kms_cookie.key_id

  depends_on = [module.kms_cookie]
}