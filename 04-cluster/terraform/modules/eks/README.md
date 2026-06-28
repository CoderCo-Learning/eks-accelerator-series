# modules/eks

Your own EKS module. No `terraform-aws-modules/eks` anywhere. The rubric wants resources you can explain, so this is the bare set and nothing more.

## What it creates

- A cluster IAM role with `AmazonEKSClusterPolicy`.
- An `aws_eks_cluster` on the version you pass, control plane logging on, `authentication_mode = API`.
- A node IAM role with the worker, ECR-read and CNI policies, plus `AmazonEBSCSIDriverPolicy` as a temporary shortcut until IRSA.
- One bootstrap managed node group, AL2023, on-demand, across the subnets you pass.
- An explicit cluster-admin access entry for every ARN in `admin_principal_arns`. The cluster creator does not get implicit admin (`bootstrap_creator_admin = false`), so access is explicit and the break-it demo genuinely locks you out. A precondition stops you applying with no admins at all.
- The four core addons as EKS-managed addons: kube-proxy, vpc-cni, coredns, aws-ebs-csi-driver.

## What it does not create, on purpose

- No VPC. You pass `subnet_ids` from the EP3 VPC module.
- No pod-level identity. That is the next session. Pod Identity is the current AWS default (a small agent addon, no OIDC provider needed), IRSA is the older OIDC-based mechanism still required for some cases. The `oidc_issuer_url` output is here so you can wire IRSA when you need it.
- No Karpenter. The bootstrap group is deliberately small, Karpenter takes over scaling later.

## Usage

```hcl
module "eks" {
  source = "../../modules/eks"

  cluster_name    = "eks-accel-dev"
  cluster_version = "1.33"
  subnet_ids      = module.vpc.private_subnet_ids

  public_access_cidrs  = ["203.0.113.10/32"] # your IP, not 0.0.0.0/0
  admin_principal_arns = ["arn:aws:iam::111122223333:role/your-sso-role"]
}
```

## The temporary EBS shortcut

`AmazonEBSCSIDriverPolicy` sits on the node role so the EBS addon works tonight. The proper home is a dedicated role only the driver's service account can assume. The current clean path is a Pod Identity association on the addon, which needs the EKS Pod Identity Agent and no OIDC provider. IRSA via `service_account_role_arn` also works. Either way you do it in the identity session, then take the policy off the node role. Note it in your project README so the live review knows you know.

## Inputs and outputs

See `variables.tf` and `outputs.tf`. Every variable has a description that explains why it is there.
