# Three AZs in London. Pulled from the provider so you do not hard-code zones
# that might not exist in another account.
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

locals {
  azs = slice(data.aws_availability_zones.available.names, 0, 3)
}

module "vpc" {
  source = "../../modules/vpc"

  name         = var.name
  cluster_name = var.cluster_name
  cidr_block   = "10.0.0.0/16"
  azs          = local.azs

  # Public subnets are tiny: they only hold the NLB and the NAT gateways.
  public_subnet_cidrs = ["10.0.96.0/24", "10.0.97.0/24", "10.0.98.0/24"]

  # Private subnets are huge: every pod takes a VPC IP under the default CNI.
  # A /19 is ~8,000 addresses per AZ. Karpenter will use them faster than you think.
  private_subnet_cidrs = ["10.0.0.0/19", "10.0.32.0/19", "10.0.64.0/19"]

  nat_mode = var.nat_mode
}
