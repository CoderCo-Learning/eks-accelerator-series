output "cluster_name" {
  description = "The cluster name. Feed it to `aws eks update-kubeconfig --name`."
  value       = aws_eks_cluster.this.name
}

output "cluster_endpoint" {
  description = "The cluster API server endpoint."
  value       = aws_eks_cluster.this.endpoint
}

output "cluster_certificate_authority_data" {
  description = "Base64 CA cert for the API server. Used by kubeconfig and any in-cluster client."
  value       = aws_eks_cluster.this.certificate_authority[0].data
}

output "cluster_security_group_id" {
  description = "The cluster security group EKS creates. Karpenter and other tools discover nodes through it later."
  value       = aws_eks_cluster.this.vpc_config[0].cluster_security_group_id
}

output "oidc_issuer_url" {
  description = "The cluster OIDC issuer URL. The IRSA session turns this into an IAM OIDC provider."
  value       = aws_eks_cluster.this.identity[0].oidc[0].issuer
}

output "node_role_arn" {
  description = "ARN of the node IAM role. Karpenter reuses it for the nodes it brings up."
  value       = aws_iam_role.node.arn
}

output "node_group_id" {
  description = "The bootstrap node group id."
  value       = aws_eks_node_group.bootstrap.id
}
