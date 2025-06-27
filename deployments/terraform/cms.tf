# https://parkjunwoo.com/microstral/deployments/terraform/cms.tf

data "aws_cloudfront_cache_policy" "caching_optimized" {
  name = "Managed-CachingOptimized"
}

locals {
  cache_policy_id = data.aws_cloudfront_cache_policy.caching_optimized.id

  cms_domain = "cms.${var.domain}"
  cms_ec2_domain = "cms-ec2.${var.domain}"
  cms_site_name = "${var.project}-cms-${var.workspace}"
  cms_bucket_name = "${var.project}-cms-${var.workspace}"
  cms_tags = {
    Project   = var.project
    Workspace = terraform.workspace
    Site = "cms"
  }
}

resource "aws_ec2_reserved_instance" "cms" {
  instance_count      = 1
  instance_type       = "t3a.micro"
  availability_zone   = "ap-northeast-2a"
  offering_class      = "standard"
  offering_type       = "No Upfront"
  product_description = "Linux/UNIX"
  term                = 31536000      # 1년

  tags = local.cms_tags
}

resource "aws_security_group" "cms" {
  name        = "${var.project}-cms-sg"
  description = "Security group for CMS EC2 instance"
  vpc_id      = aws_vpc.main.id

  ingress {
    description = "Allow HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "Allow HTTPS"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # SSH 접근 (선택: 내 IP만 허용)
  ingress {
    description = "Allow SSH from my IP"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["${chomp(data.http.my_ip.body)}/32"]
  }

  # 아웃바운드: 모든 외부로 나가는 요청 허용 (기본값)
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = local.cms_tags
}

resource "tls_private_key" "cms" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_key_pair" "cms" {
  key_name = "${var.project}-${terraform.workspace}"
  public_key = tls_private_key.cms.public_key_openssh
  tags = {
    Project   = var.project
    Workspace = terraform.workspace
    Site = "cms"
  }
}

resource "local_file" "private_key" {
  content  = tls_private_key.cms.private_key_pem
  filename = "${path.module}/${var.project}.pem"
  file_permission = "0400"
}

resource "aws_instance" "cms" {
  ami                         = "ami-0662f4965dfc70aca"
  instance_type               = "t3a.micro"
  subnet_id                   = aws_subnet.public_a.id
  vpc_security_group_ids      = [aws_security_group.cms.id]
  associate_public_ip_address = true
  key_name                    = aws_key_pair.cms.key_name

  tags = local.cms_tags
}

# 사이트 ACM 인증서 생성
module "acm_site_cms" {
  source  = "terraform-aws-modules/acm/aws"
  version = "~> 5.1.1"
  providers = {
    aws = aws.us_east_1
  }
  domain_name       = local.cms_domain
  validation_method = "DNS"
  zone_id           = local.zone_id
  tags = local.cms_tags
}

# 버킷 생성
module "bucket_cms" {
  source  = "terraform-aws-modules/s3-bucket/aws"
  version = "4.7.0"
  providers = {
    aws = aws
  }
  bucket        = local.cms_bucket_name
  versioning    = { enabled = false }
  force_destroy = false
  website = {
    index_document = "index.html"
    error_document = "404.html"
  }
  tags = local.cms_tags
}

# CloudFront Public Key 등록
resource "aws_cloudfront_public_key" "public_key_cms" {
  provider = aws.us_east_1
  name        = "${local.cms_site_name}-publickey"
  encoded_key = data.aws_kms_public_key.cookie.public_key_pem
}

# CloudFront Key Group 생성
resource "aws_cloudfront_key_group" "key_group_cms" {
  provider = aws.us_east_1
  name  = "${local.cms_site_name}-keygroup"
  items = [aws_cloudfront_public_key.public_key_cms.id]
}

# OAC (Origin Access Control) 생성
resource "aws_cloudfront_origin_access_control" "oac_cms" {
  provider = aws.us_east_1
  name                              = "${local.cms_site_name}-oac"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_wafv2_web_acl" "cms_cdn_acl" {
  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "cms-cdn-waf"
    sampled_requests_enabled   = true
  }
  provider = aws.us_east_1

  name  = "${local.cms_site_name}-waf"
  scope = "CLOUDFRONT"

  default_action {
    allow {}
  }

  rule {
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "cms-limit-by-ip"
      sampled_requests_enabled   = true
    }

    name     = "limit-by-ip"
    priority = 1
    action {
       block {} 
    }

    statement {
      rate_based_statement {
        limit              = 500
        aggregate_key_type = "IP"
      }
    }
  }
  tags = local.cms_tags
}

# CloudFront Distribution 생성
resource "aws_cloudfront_distribution" "cdn_cms" {
  provider = aws.us_east_1
  enabled             = true
  comment             = local.cms_site_name
  default_root_object = "index.html"
  aliases             = [local.cms_domain]
  tags                = local.cms_tags
  web_acl_id           = aws_wafv2_web_acl.cms_cdn_acl.arn

  origin {
    origin_id                = local.cms_site_name
    domain_name              = module.bucket_cms.s3_bucket_bucket_regional_domain_name
    origin_access_control_id = aws_cloudfront_origin_access_control.oac_cms.id
  }

  origin {
    origin_id   = "EC2Origin"
    domain_name = local.cms_ec2_domain

    custom_origin_config {
      origin_protocol_policy = "https-only"
      http_port              = 80
      https_port             = 443
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = local.cms_site_name
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD", "OPTIONS"]
    compress               = true
    cache_policy_id     = local.cache_policy_id
  }

  ordered_cache_behavior {
    path_pattern           = "/app/*"
    target_origin_id       = local.cms_site_name
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD", "OPTIONS"]
    compress               = true
    cache_policy_id     = local.cache_policy_id
    trusted_key_groups = [aws_cloudfront_key_group.key_group_cms.id]
  }

  ordered_cache_behavior {
    path_pattern           = "/api/*"
    target_origin_id       = "EC2Origin"
    viewer_protocol_policy = "https-only"
    allowed_methods        = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    forwarded_values {
      query_string = true
      headers      = ["Authorization"]

      cookies {
        forward = "whitelist"
        whitelisted_names = ["t", "r"]
      }
    }
  }

  viewer_certificate {
    acm_certificate_arn = module.acm_site_cms.acm_certificate_arn
    ssl_support_method  = "sni-only"
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  custom_error_response {
    error_code            = 403
    response_page_path    = "/404.html"
    response_code         = 200
    error_caching_min_ttl = 300
  }
  custom_error_response {
    error_code            = 404
    response_page_path    = "/404.html"
    response_code         = 200
    error_caching_min_ttl = 300
  }
  custom_error_response {
    error_code            = 500
    response_page_path    = "/404.html"
    response_code         = 200
    error_caching_min_ttl = 300
  }
}

resource "aws_route53_record" "cms" {
  zone_id = local.zone_id
  name    = local.cms_domain
  type    = "A"
  alias {
    name                   = aws_cloudfront_distribution.cdn_cms.domain_name
    zone_id                = aws_cloudfront_distribution.cdn_cms.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "cms_ec2" {
  zone_id = local.zone_id
  name    = local.cms_ec2_domain
  type    = "A"
  ttl     = 300

  records = [aws_instance.cms.public_ip]
}

resource "aws_s3_bucket_policy" "bucket_cms_oac" {
  bucket = module.bucket_cms.s3_bucket_id

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Sid       = "AllowCloudFrontOACRead"
        Effect    = "Allow"
        Principal = {
          Service = "cloudfront.amazonaws.com"
        }
        Action    = ["s3:GetObject"]
        Resource  = "${module.bucket_cms.s3_bucket_arn}/*"
        Condition = {
          StringEquals = {
            "AWS:SourceArn" = aws_cloudfront_distribution.cdn_cms.arn
          }
        }
      }
    ]
  })
}
