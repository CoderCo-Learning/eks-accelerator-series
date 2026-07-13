terraform {
  required_version = ">= 1.10"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    # Helm installs the Karpenter controller. Pin the major version.
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.17"
    }
  }
}
