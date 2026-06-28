variable "region" {
  description = "AWS region. The series uses London."
  type        = string
  default     = "eu-west-2"
}

variable "cluster_name" {
  description = "EKS cluster name. Must match the cluster_name you used on the EP3 VPC so the subnet tags line up."
  type        = string
  default     = "eks-accel-dev"
}

variable "cluster_version" {
  description = "Kubernetes version. Project floor is 1.33, but that version is leaving standard support around mid-2026. Default is a current standard-support version. Pick the newest standard-support version for real use."
  type        = string
  default     = "1.34"
}

variable "private_subnet_ids" {
  description = "The three private subnet ids from `terraform output private_subnet_ids` in EP3."
  type        = list(string)
}

variable "public_access_cidrs" {
  description = "CIDRs allowed to reach the public API endpoint. Set this to your own /32. Do not leave it open."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "admin_principal_arns" {
  description = "IAM principal ARNs that get explicit cluster admin. Put your own `aws sts get-caller-identity` ARN here."
  type        = list(string)
  default     = []
}
