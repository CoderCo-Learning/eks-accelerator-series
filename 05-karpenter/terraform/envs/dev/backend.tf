# Remote state, same pattern as EP4. Create the bucket once, then turn this on.
# use_lockfile needs Terraform >= 1.10.
#
# terraform {
#   backend "s3" {
#     bucket       = "eks-accel-tfstate-<your-account-id>"
#     key          = "dev/karpenter/terraform.tfstate"
#     region       = "eu-west-2"
#     use_lockfile = true
#     encrypt      = true
#   }
# }
