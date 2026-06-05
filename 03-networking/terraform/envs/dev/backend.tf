# Remote state is a project hard requirement. Create the bucket and lock table
# once, by hand or in a tiny bootstrap config, then point this at them.
# Left commented so `terraform init` works locally before you have a backend.
#
# terraform {
#   backend "s3" {
#     bucket       = "eks-accel-tfstate-<your-account-id>"
#     key          = "dev/networking/terraform.tfstate"
#     region       = "eu-west-2"
#     use_lockfile = true
#     encrypt      = true
#   }
# }
