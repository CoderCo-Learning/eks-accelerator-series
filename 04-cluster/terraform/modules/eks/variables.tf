variable "cluster_name" {
  description = "Name of the EKS cluster. Must match the kubernetes.io/cluster/<name> tag on the subnets from EP3."
  type        = string
}

variable "cluster_version" {
  description = "Kubernetes minor version. The project floor is 1.33, but 1.33 reaches end of standard support around mid-2026 and then bills at the extended-support rate. Default to a current standard-support version and track the AWS support calendar."
  type        = string
  default     = "1.34"
}

variable "subnet_ids" {
  description = "Private subnet ids from the EP3 VPC, one per AZ. The control plane ENIs and the nodes both land here."
  type        = list(string)
}

variable "endpoint_public_access" {
  description = "Expose the cluster API endpoint to the internet. Keep it on for a laptop-driven dev cluster, but lock public_access_cidrs."
  type        = bool
  default     = true
}

variable "endpoint_private_access" {
  description = "Expose the cluster API endpoint inside the VPC. Always on, it is how nodes reach the control plane without leaving the VPC."
  type        = bool
  default     = true
}

variable "public_access_cidrs" {
  description = "CIDRs allowed to hit the public endpoint. Never leave this at 0.0.0.0/0. Set it to your own /32."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "enabled_log_types" {
  description = "Control plane log types shipped to CloudWatch. All five is the project default, you want the audit log for the live review."
  type        = list(string)
  default     = ["api", "audit", "authenticator", "controllerManager", "scheduler"]
}

variable "node_instance_types" {
  description = "Instance types for the bootstrap node group. One small on-demand type is right, Karpenter handles real capacity later."
  type        = list(string)
  default     = ["t3.large"]
}

variable "node_desired_size" {
  description = "Desired node count for the bootstrap group. Two covers the system pods across two AZs."
  type        = number
  default     = 2
}

variable "node_min_size" {
  description = "Minimum node count for the bootstrap group."
  type        = number
  default     = 2
}

variable "node_max_size" {
  description = "Maximum node count for the bootstrap group. Keep it low, this group is not where you scale."
  type        = number
  default     = 3
}

variable "bootstrap_creator_admin" {
  description = "Whether the IAM principal that runs apply gets implicit cluster admin. Default false so access is explicit and auditable through access entries. Leaving it true AND listing the creating ARN in admin_principal_arns throws a 409 ResourceInUseException, because EKS already made that entry."
  type        = bool
  default     = false
}

variable "admin_principal_arns" {
  description = "IAM principal ARNs that get an explicit cluster-admin access entry. With bootstrap_creator_admin = false you must list at least one reachable identity here (your `aws sts get-caller-identity` ARN) or nobody can use the cluster. Do not list the creating ARN while bootstrap_creator_admin is true."
  type        = list(string)
  default     = []
}

variable "addon_versions" {
  description = "Optional pinned versions per addon, keyed by addon name (vpc-cni, coredns, kube-proxy, aws-ebs-csi-driver). Leave a key out to let EKS pick the default compatible version."
  type        = map(string)
  default     = {}
}

variable "tags" {
  description = "Module-local tags. Provider default_tags is preferred for the common set."
  type        = map(string)
  default     = {}
}
