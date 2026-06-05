# Module: vpc

A VPC built for EKS rather than a generic one. Three AZs, public subnets for the NLB and NAT, large private subnets for nodes and pods, subnet tags the AWS Load Balancer Controller looks for and a configurable egress story (NAT, endpoints or both).

This is reference. Read it, understand every resource, then write your own in your project repo. Wrapping `terraform-aws-modules/vpc` does not count for the live review.

## Why these choices

- **Private subnets are large, public subnets are small.** The AWS VPC CNI hands every pod a real VPC IP out of the node's subnet. Public subnets only ever hold the NLB and the NAT gateways, so a `/24` each is plenty. See the CIDR maths in the [episode README](../../../README.md).
- **Subnet tags drive discovery.** `kubernetes.io/role/elb` on public subnets and `kubernetes.io/role/internal-elb` on private ones tell the load balancer controller where to put internet-facing and internal load balancers. Miss them and your `Service` of type LoadBalancer sits in `pending` forever.
- **`nat_mode` is a real lever.** `single` is one NAT for the whole VPC: cheap, not highly available. `per_az` is one per zone: survives an AZ outage, costs roughly three times as much. `none` leans entirely on endpoints and will not reach third-party APIs like Twilio or a carrier. Pick deliberately and defend it.
- **The S3 gateway endpoint is always on.** It is free and it carries ECR image-layer pulls, which is the heaviest path off your nodes once Karpenter starts churning.

## Usage

```hcl
module "vpc" {
  source = "../../modules/vpc"

  name         = "eks-accel-dev"
  cluster_name = "eks-accel-dev"
  cidr_block   = "10.0.0.0/16"
  azs          = ["eu-west-2a", "eu-west-2b", "eu-west-2c"]

  public_subnet_cidrs  = ["10.0.96.0/24", "10.0.97.0/24", "10.0.98.0/24"]
  private_subnet_cidrs = ["10.0.0.0/19", "10.0.32.0/19", "10.0.64.0/19"]

  nat_mode = "single"
}
```

## Inputs

| Name | Description | Default |
|---|---|---|
| `name` | Resource name prefix | n/a |
| `cluster_name` | EKS cluster name for the discovery tag | n/a |
| `cidr_block` | VPC CIDR | `10.0.0.0/16` |
| `azs` | AZs to spread across | n/a |
| `public_subnet_cidrs` | One CIDR per AZ, small | n/a |
| `private_subnet_cidrs` | One CIDR per AZ, large | n/a |
| `nat_mode` | `none`, `single` or `per_az` | `single` |
| `interface_endpoints` | Service short names for PrivateLink endpoints | ECR, STS, SSM, Secrets Manager, logs |
| `tags` | Extra tags | `{}` |

## Outputs

| Name | Description |
|---|---|
| `vpc_id` | VPC id |
| `vpc_cidr_block` | VPC CIDR |
| `public_subnet_ids` | Public subnet ids |
| `private_subnet_ids` | Private subnet ids, feed these to EKS |
| `nat_gateway_ids` | NAT gateway ids (empty when `nat_mode = none`) |
| `interface_endpoint_ids` | Map of service name to endpoint id |
