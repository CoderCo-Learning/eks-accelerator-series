output "oidc_provider_arn" {
  description = "ARN of the IAM OIDC provider. Every IRSA role in the cluster trusts this."
  value       = aws_iam_openid_connect_provider.this.arn
}

output "ebs_csi_role_arn" {
  description = "Role ARN for the EBS driver. Set this as service_account_role_arn on the EP4 aws_eks_addon.ebs_csi, then remove AmazonEBSCSIDriverPolicy from the node role."
  value       = aws_iam_role.ebs_csi.arn
}

output "demo_role_arn" {
  description = "Role ARN for the IRSA demo. Put it in the irsa-demo service account annotation in k8s/irsa-demo.yaml."
  value       = aws_iam_role.demo.arn
}
