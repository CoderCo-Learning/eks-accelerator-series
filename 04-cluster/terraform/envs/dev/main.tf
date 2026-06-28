# The cluster goes in the private subnets you built in EP3.
#
# Two ways to get those subnet ids in here:
#   1. Paste them into terraform.tfvars (simple, what the example shows).
#   2. Read them from EP3 remote state once both envs are on an S3 backend:
#
#      data "terraform_remote_state" "vpc" {
#        backend = "s3"
#        config = {
#          bucket = "eks-accel-tfstate-<your-account-id>"
#          key    = "dev/networking/terraform.tfstate"
#          region = "eu-west-2"
#        }
#      }
#      # then: subnet_ids = data.terraform_remote_state.vpc.outputs.private_subnet_ids

module "eks" {
  source = "../../modules/eks"

  cluster_name    = var.cluster_name
  cluster_version = var.cluster_version
  subnet_ids      = var.private_subnet_ids

  # Lock the public endpoint to your own IP. Never ship 0.0.0.0/0.
  public_access_cidrs = var.public_access_cidrs

  # Your human ARN gets an explicit cluster-admin entry. This is the line that
  # guarantees you can reach the cluster even when CI is the creator.
  admin_principal_arns = var.admin_principal_arns
}
