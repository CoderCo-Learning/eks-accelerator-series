output "cluster_name" {
  value = module.eks.cluster_name
}

output "cluster_endpoint" {
  value = module.eks.cluster_endpoint
}

output "oidc_issuer_url" {
  description = "Hand this to the IRSA session next week."
  value       = module.eks.oidc_issuer_url
}

output "node_role_arn" {
  description = "Karpenter reuses this node role later."
  value       = module.eks.node_role_arn
}

output "update_kubeconfig" {
  description = "Copy-paste this to point kubectl at the new cluster."
  value       = "aws eks update-kubeconfig --name ${module.eks.cluster_name} --region ${var.region}"
}
