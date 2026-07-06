output "controller_role_arn" {
  value = module.karpenter.controller_role_arn
}

output "interruption_queue_name" {
  value = module.karpenter.interruption_queue_name
}

output "node_role_name" {
  description = "Put this in k8s/ec2nodeclass.yaml spec.role."
  value       = module.karpenter.node_role_name
}

output "discovery_tag" {
  description = "Put this in the EC2NodeClass subnet and security-group selectors."
  value       = module.karpenter.discovery_tag
}
