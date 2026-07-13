variable "cluster_name" {
  description = "The EKS cluster name from EP4. Used for discovery tags, the controller trust policy and the Helm settings."
  type        = string
}

variable "node_role_arn" {
  description = "ARN of the EP4 node role. Karpenter nodes reuse it, so they inherit its existing EC2_LINUX access entry and join with nothing extra. The controller gets iam:PassRole on it."
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet ids from EP3. Tagged with karpenter.sh/discovery so the EC2NodeClass can find them."
  type        = list(string)
}

variable "cluster_security_group_id" {
  description = "The cluster security group id from EP4. Tagged with karpenter.sh/discovery so Karpenter nodes attach the right SG."
  type        = string
}

variable "karpenter_version" {
  description = "Karpenter Helm chart version. v1 line. Check for the latest patch before you pin."
  type        = string
  default     = "1.13.0"
}

variable "namespace" {
  description = "Namespace for the Karpenter controller. kube-system keeps it alongside the other system components."
  type        = string
  default     = "kube-system"
}

variable "service_account" {
  description = "Service account name the Helm chart creates and the Pod Identity association binds. Leave as karpenter unless you have a reason."
  type        = string
  default     = "karpenter"
}

variable "controller_replicas" {
  description = "Karpenter controller replicas. Two for leader-elected high availability on the bootstrap group."
  type        = number
  default     = 2
}

variable "tags" {
  description = "Module-local tags. Provider default_tags is preferred for the common set."
  type        = map(string)
  default     = {}
}
