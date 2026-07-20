# Remote state, same pattern as the earlier sessions. use_lockfile needs TF >= 1.10.
#
# terraform {
#   backend "s3" {
#     bucket       = "eks-accel-tfstate-<your-account-id>"
#     key          = "dev/storage/terraform.tfstate"
#     region       = "eu-west-2"
#     use_lockfile = true
#     encrypt      = true
#   }
# }
