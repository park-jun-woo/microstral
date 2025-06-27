# https://parkjunwoo.com/microstral/deployments/terraform/main.tf

terraform {
  required_version = ">= 1.5.0"

  backend "s3" {}

  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 5.95.0" }
    tls = { source = "hashicorp/tls", version = "~> 4.0" }
  }
}

provider "aws" {
  region = var.aws_region
}

provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"
}

provider "tls" {}

data "http" "my_ip" {
  url = "https://checkip.amazonaws.com"
}

locals {
  zone_id = module.route53_zones.route53_zone_zone_id[var.domain]
  tags         = {
    Project   = var.project
    Workspace = terraform.workspace
  }
}

# Route53 Hosted Zone 생성
module "route53_zones" {
  source  = "terraform-aws-modules/route53/aws//modules/zones"
  version = "5.0.0"

  zones = {
    "${var.domain}" = {
      name         = var.domain
      comment      = "Public hosted zone for ${var.domain}"
      private_zone = false
      tags         = local.tags
    }
  }
}

resource "aws_route53_record" "alias_gmail" {
  zone_id = local.zone_id
  name    = var.domain
  type    = "MX"
  ttl     = 86400*7

  records = [
    "1 ASPMX.L.GOOGLE.COM.",
    "5 ALT1.ASPMX.L.GOOGLE.COM.",
    "5 ALT2.ASPMX.L.GOOGLE.COM.",
    "10 ALT3.ASPMX.L.GOOGLE.COM.",
    "10 ALT4.ASPMX.L.GOOGLE.COM.",
  ]
}