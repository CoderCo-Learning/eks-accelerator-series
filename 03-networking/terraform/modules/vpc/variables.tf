variable "name" {
  description = "Name prefix for every resource in this module, e.g. eks-accel-dev."
  type        = string
}

variable "cluster_name" {
  description = "EKS cluster name this VPC is built for. Used for the kubernetes.io/cluster/<name> subnet discovery tag."
  type        = string
}

variable "cidr_block" {
  description = "The VPC CIDR. A /16 gives you ~65k addresses. With the AWS VPC CNI every pod takes one, so do not go smaller."
  type        = string
  default     = "10.0.0.0/16"
}

variable "azs" {
  description = "Availability zones to spread subnets across. Three is the project minimum."
  type        = list(string)
}

variable "public_subnet_cidrs" {
  description = "One CIDR per AZ for the public subnets. These hold the NLB and the NAT gateways, nothing else. Keep them small."
  type        = list(string)
}

variable "private_subnet_cidrs" {
  description = "One CIDR per AZ for the private subnets. Nodes and pods live here, so these must be large."
  type        = list(string)
}

variable "nat_mode" {
  description = "How outbound internet egress is provided. none = endpoints only, single = one shared NAT (cheap, not HA), per_az = one NAT per AZ (HA, ~3x the cost)."
  type        = string
  default     = "single"

  validation {
    condition     = contains(["none", "single", "per_az"], var.nat_mode)
    error_message = "nat_mode must be one of: none, single, per_az."
  }
}

variable "interface_endpoints" {
  description = "AWS service short names to create interface (PrivateLink) endpoints for. Empty list disables them. The S3 gateway endpoint is always created because it is free."
  type        = list(string)
  default = [
    "ecr.api",
    "ecr.dkr",
    "sts",
    "ssm",
    "ssmmessages",
    "ec2messages",
    "secretsmanager",
    "logs",
  ]
}

variable "tags" {
  description = "Tags applied to every resource. default_tags on the provider is preferred, this is here for module-local additions."
  type        = map(string)
  default     = {}
}
