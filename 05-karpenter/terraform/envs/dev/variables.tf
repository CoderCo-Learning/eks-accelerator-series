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

variable "node_role_arn" {
  description = "EP4 node role ARN. `terraform output node_role_arn` in the EP4 dev env."
  type        = string
}

variable "private_subnet_ids" {
  description = "EP3 private subnet ids. `terraform output private_subnet_ids` in the EP3 dev env."
  type        = list(string)
}

variable "cluster_security_group_id" {
  description = "EP4 cluster security group id. `terraform output cluster_security_group_id`."
  type        = string
}

variable "karpenter_version" {
  description = "Karpenter Helm chart version, v1 line. Check for the latest patch."
  type        = string
  default     = "1.13.0"
}
