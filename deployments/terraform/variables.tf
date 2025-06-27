# https://parkjunwoo.com/microstral/deployments/terraform/variables.tf

variable "project" {
  description = "프로젝트명, 예: sample"
  type        = string
}

variable "workspace" {
  description = "워크스페이스명 (dev, prod 등)"
  type        = string
}

variable "aws_region" {
  description = "예: ap-northeast-2"
  type        = string
}

variable "domain" {
  description = "루트 도메인, 예: example.com"
  type        = string
}