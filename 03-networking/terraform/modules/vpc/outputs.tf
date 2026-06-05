output "vpc_id" {
  description = "The VPC id."
  value       = aws_vpc.this.id
}

output "vpc_cidr_block" {
  description = "The VPC CIDR."
  value       = aws_vpc.this.cidr_block
}

output "public_subnet_ids" {
  description = "Public subnet ids, one per AZ. Feed these to the NLB."
  value       = aws_subnet.public[*].id
}

output "private_subnet_ids" {
  description = "Private subnet ids, one per AZ. Feed these to the EKS cluster and node groups."
  value       = aws_subnet.private[*].id
}

output "nat_gateway_ids" {
  description = "NAT gateway ids. Empty when nat_mode is none."
  value       = aws_nat_gateway.this[*].id
}

output "interface_endpoint_ids" {
  description = "Map of service short name to interface endpoint id."
  value       = { for k, v in aws_vpc_endpoint.interface : k => v.id }
}
