terraform {
  # 1.10+ for native S3 state locking (use_lockfile), which replaces the old DynamoDB lock table.
  required_version = ">= 1.10"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

provider "aws" {
  region = var.region

  default_tags {
    tags = {
      Project   = "eks-accelerator"
      Env       = "dev"
      ManagedBy = "terraform"
    }
  }
}
