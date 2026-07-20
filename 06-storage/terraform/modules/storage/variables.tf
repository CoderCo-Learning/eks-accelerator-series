variable "cluster_name" {
  description = "The EKS cluster name from EP4. Used to name the roles."
  type        = string
}

variable "oidc_issuer_url" {
  description = "The cluster OIDC issuer URL from EP4 (`terraform output oidc_issuer_url`). This is what the IRSA trust policies point at."
  type        = string
}

variable "namespace" {
  description = "Namespace the EBS driver and the demo run in."
  type        = string
  default     = "kube-system"
}

variable "ebs_csi_service_account" {
  description = "Service account the EBS CSI controller uses. The trust policy is scoped to exactly this name."
  type        = string
  default     = "ebs-csi-controller-sa"
}

variable "ebs_csi_policy_arn" {
  description = "Managed policy for the EBS driver. V2 is the current, tighter version. V1 is arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy."
  type        = string
  default     = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicyV2"
}

variable "demo_service_account" {
  description = "Service account for the IRSA break-it demo. Change it to a wrong name and re-apply to watch the trust fail."
  type        = string
  default     = "irsa-demo"
}

variable "tags" {
  description = "Module-local tags. Provider default_tags is preferred for the common set."
  type        = map(string)
  default     = {}
}
