output "vpc_id" {
  value = module.vpc.vpc_id
}

output "private_subnet_ids" {
  description = "Hand these to the EKS module in EP4."
  value       = module.vpc.private_subnet_ids
}

output "public_subnet_ids" {
  value = module.vpc.public_subnet_ids
}

output "nat_gateway_ids" {
  value = module.vpc.nat_gateway_ids
}
