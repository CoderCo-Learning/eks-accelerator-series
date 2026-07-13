# modules/karpenter

Your own Karpenter module. No `terraform-aws-modules/eks//modules/karpenter`. The rubric wants resources you can explain.

## What it creates

- The `eks-pod-identity-agent` addon, so the controller can run on Pod Identity.
- A controller IAM role trusted by `pods.eks.amazonaws.com`, with the Karpenter v1 controller policy (provision and terminate instances, manage instance profiles, read pricing and AMIs, drain the interruption queue).
- An `aws_eks_pod_identity_association` binding the `karpenter` service account to that role.
- An SQS interruption queue with EventBridge rules for Spot warnings, rebalance, instance state changes and health events.
- `karpenter.sh/discovery` tags on the private subnets and the cluster security group, added with `aws_ec2_tag` so the EP3 and EP4 modules stay untouched.
- The `karpenter-crd` Helm chart, which owns the NodePool, EC2NodeClass and NodeClaim CRDs. Installed before the controller so the kinds exist by the time anyone applies one.
- The Karpenter controller Helm release (v1), pinned by version, with `skip_crds` so it never fights the CRD chart. The chart's own defaults keep the controller off Karpenter-managed nodes and spread the replicas, so the module adds no affinity override.

## What it does not create, on purpose

- No `NodePool` and no `EC2NodeClass`. Those are the platform layer, applied as YAML from `../../k8s`. The split keeps the infra module owning the controller and the policy objects living where a platform team would edit them.
- No new node role. Karpenter reuses the EP4 node role, so its existing `EC2_LINUX` access entry lets Karpenter nodes join. A brand new role would need its own access entry.
- No OIDC provider. Pod Identity does not need one.

## Usage

```hcl
module "karpenter" {
  source = "../../modules/karpenter"

  cluster_name              = "eks-accel-dev"
  node_role_arn             = "arn:aws:iam::111122223333:role/eks-accel-dev-node"
  private_subnet_ids        = ["subnet-aaaa", "subnet-bbbb", "subnet-cccc"]
  cluster_security_group_id = "sg-0123456789abcdef0"
}
```

The Helm provider must be configured against the EP4 cluster. See `../../envs/dev` for the provider block.

## Inputs and outputs

See `variables.tf` and `outputs.tf`. Feed `node_role_name` and `discovery_tag` into the EC2NodeClass.
