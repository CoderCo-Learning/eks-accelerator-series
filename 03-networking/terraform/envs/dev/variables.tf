variable "region" {
  description = "AWS region. The series uses London."
  type        = string
  default     = "eu-west-2"
}

variable "name" {
  description = "Name prefix for the environment."
  type        = string
  default     = "eks-accel-dev"
}

variable "cluster_name" {
  description = "EKS cluster name this VPC is built for. Must match the cluster you create in EP4."
  type        = string
  default     = "eks-accel-dev"
}

variable "nat_mode" {
  description = "none, single or per_az. Dev uses single to keep the bill down. Defend whatever you pick."
  type        = string
  default     = "single"
}
