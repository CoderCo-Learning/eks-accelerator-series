output "oidc_provider_arn" {
  value = module.storage.oidc_provider_arn
}

output "ebs_csi_role_arn" {
  description = "Set this as service_account_role_arn on the EP4 EBS addon."
  value       = module.storage.ebs_csi_role_arn
}

output "demo_role_arn" {
  description = "Put this in the irsa-demo service account annotation in k8s/irsa-demo.yaml."
  value       = module.storage.demo_role_arn
}
