# modules/storage

Your own storage IAM module. No upstream module for the OIDC provider or the roles, because understanding the IRSA trust chain is the whole point of the session.

## What it creates

- An IAM OIDC provider for the cluster, the thing that lets AWS trust tokens the cluster issues.
- An EBS CSI driver role, trusted only by the `ebs-csi-controller-sa` service account, carrying `AmazonEBSCSIDriverPolicyV2`.
- A throwaway demo role for the break-the-trust deep dive, trusted by an `irsa-demo` service account, with no policy attached because `sts:get-caller-identity` needs none.

## What it does not create, on purpose

- No StorageClass, VolumeSnapshotClass or PVC. Those are Kubernetes objects, applied from `../../k8s`.
- No change to the EP4 addon. You wire `ebs_csi_role_arn` onto the EP4 `aws_eks_addon.ebs_csi` yourself and remove the node-role shortcut. That edit is the point where the shortcut becomes the real thing.

## Usage

```hcl
module "storage" {
  source = "../../modules/storage"

  cluster_name    = "eks-accel-dev"
  oidc_issuer_url = "https://oidc.eks.eu-west-2.amazonaws.com/id/EXAMPLE"
}
```

## Wiring the driver after apply

1. In EP4, set `service_account_role_arn = "<ebs_csi_role_arn>"` on `aws_eks_addon.ebs_csi`.
2. In EP4, remove the `AmazonEBSCSIDriverPolicy` attachment from the node role.
3. `terraform apply` EP4. The EBS controller restarts on its own role.

## Breaking the trust (deep dive)

Change `demo_service_account` to a name that does not match the `irsa-demo` service account, run `terraform apply`, then re-run the demo job. The assume fails with `AccessDenied` on `sts:AssumeRoleWithWebIdentity`. Change it back to fix it.
