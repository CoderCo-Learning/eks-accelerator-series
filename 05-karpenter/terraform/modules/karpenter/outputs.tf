output "controller_role_arn" {
  description = "ARN of the Karpenter controller role, bound to the karpenter service account through Pod Identity."
  value       = aws_iam_role.controller.arn
}

output "interruption_queue_name" {
  description = "SQS interruption queue name. Passed to the Helm chart as settings.interruptionQueue."
  value       = aws_sqs_queue.interruption.name
}

output "node_role_name" {
  description = "The node role name to put in the EC2NodeClass spec.role."
  value       = local.node_role
}

output "discovery_tag" {
  description = "The discovery tag value to use in the EC2NodeClass subnet and security-group selectors."
  value       = var.cluster_name
}
