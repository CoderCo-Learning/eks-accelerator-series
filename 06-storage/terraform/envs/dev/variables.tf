variable "region" {
  description = "AWS region. The series uses London."
  type        = string
  default     = "eu-west-2"
}

variable "cluster_name" {
  description = "EKS cluster name from EP4."
  type        = string
  default     = "eks-accel-dev"
}

variable "oidc_issuer_url" {
  description = "Cluster OIDC issuer URL. `terraform output oidc_issuer_url` in the EP4 dev env."
  type        = string
}
