provider "aws" {
  region = "eu-west-2"
}

resource "random_id" "bucket_suffix" {
  byte_length = 4
}

resource "aws_s3_bucket" "image-storage-bucket" {
  bucket = "image-storage-bucket-${random_id.bucket_suffix.hex}"
}

resource "aws_iam_role" "post_image_lambda_role" {
  name               = "post_image_lambda_role"
  assume_role_policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Action": "sts:AssumeRole",
        "Principal": {
          "Service": "lambda.amazonaws.com"
        },
        "Effect": "Allow"
      }
    ]
  }
EOF
}

resource "aws_iam_policy" "post_image_lambda_policy" {
  name_prefix = "post_image_lambda-policy-"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "s3:PutObject",
        ]
        Resource = "arn:aws:s3:::*"
        Effect   = "Allow"
      },
      {
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ],
        Resource = "arn:aws:logs:*:*:*"
        Effect   = "Allow"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "post_image_iam_role_policy_attachment" {
  role       = aws_iam_role.post_image_lambda_role.name
  policy_arn = aws_iam_policy.post_image_lambda_policy.arn
}

data "archive_file" "zip_the_go_bin_post" {
  type        = "zip"
  source_dir  = "${path.module}/lambdas/image_put"
  output_path = "${path.module}/lambdas/image_put/image_put.zip"
}

resource "aws_lambda_function" "post_image_lambda_func" {
  filename         = data.archive_file.zip_the_go_bin_post.output_path
  function_name    = "Post-Image-Lambda"
  role             = aws_iam_role.post_image_lambda_role.arn
  handler          = "image_put"
  runtime          = "go1.x"
  depends_on       = [aws_iam_role_policy_attachment.post_image_iam_role_policy_attachment]
  source_code_hash = data.archive_file.zip_the_go_bin_post.output_base64sha256

  environment {
    variables = {
      S3_BUCKET_NAME = aws_s3_bucket.image-storage-bucket.bucket
    }
  }
}

resource "aws_iam_role" "get_image_lambda_role" {
  name = "get_image_lambda_role"

  assume_role_policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Action": "sts:AssumeRole",
        "Principal": {
          "Service": "lambda.amazonaws.com"
        },
        "Effect": "Allow"
      }
    ]
}
EOF
}

resource "aws_iam_policy" "get_image_lambda_policy" {
  name = "get_image_lambda_policy"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "s3:GetObject",
          "s3:ListBucket"
        ]
        Resource = "arn:aws:s3:::*"
        Effect   = "Allow"
      },
      {
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ],
        Resource = "arn:aws:logs:*:*:*"
        Effect   = "Allow"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "get_image_iam_role_policy_attachment" {
  role       = aws_iam_role.get_image_lambda_role.name
  policy_arn = aws_iam_policy.get_image_lambda_policy.arn
}

data "archive_file" "zip_the_go_bin_get" {
  type        = "zip"
  source_dir  = "${path.module}/lambdas/image_get"
  output_path = "${path.module}/lambdas/image_get/image_get.zip"
}

resource "aws_lambda_function" "get_image_lambda_func" {
  filename         = data.archive_file.zip_the_go_bin_get.output_path
  function_name    = "Get-Image-Lambda"
  role             = aws_iam_role.get_image_lambda_role.arn
  handler          = "image_get"
  runtime          = "go1.x"
  depends_on       = [aws_iam_role_policy_attachment.get_image_iam_role_policy_attachment]
  source_code_hash = data.archive_file.zip_the_go_bin_get.output_base64sha256

  environment {
    variables = {
      S3_BUCKET_NAME = aws_s3_bucket.image-storage-bucket.bucket
    }
  }
}

resource "aws_lambda_function_url" "post_function" {
  function_name      = aws_lambda_function.post_image_lambda_func.function_name
  authorization_type = "NONE"
}

resource "aws_lambda_function_url" "get_function" {
  function_name      = aws_lambda_function.get_image_lambda_func.function_name
  authorization_type = "NONE"
}

terraform {
  cloud {
    organization = "tf-org-jgcrew"
    workspaces {
      name = "mediafly-image-processor"
    }
  }
}
