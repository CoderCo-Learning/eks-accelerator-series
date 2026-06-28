###############################################################################
# Cluster IAM role. Assumed by the EKS service, not by you and not by the nodes.
# Lets the control plane manage ENIs and load balancers on your behalf.
###############################################################################
data "aws_iam_policy_document" "cluster_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["eks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "cluster" {
  name               = "${var.cluster_name}-cluster"
  assume_role_policy = data.aws_iam_policy_document.cluster_assume.json
  tags               = var.tags
}

resource "aws_iam_role_policy_attachment" "cluster" {
  role       = aws_iam_role.cluster.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
}

###############################################################################
# The cluster. authentication_mode = API means access entries only, no aws-auth.
###############################################################################
resource "aws_eks_cluster" "this" {
  name     = var.cluster_name
  version  = var.cluster_version
  role_arn = aws_iam_role.cluster.arn

  enabled_cluster_log_types = var.enabled_log_types

  access_config {
    authentication_mode                         = "API"
    bootstrap_cluster_creator_admin_permissions = var.bootstrap_creator_admin
  }

  vpc_config {
    subnet_ids              = var.subnet_ids
    endpoint_public_access  = var.endpoint_public_access
    endpoint_private_access = var.endpoint_private_access
    public_access_cidrs     = var.public_access_cidrs
  }

  tags = var.tags

  lifecycle {
    # Stop you shipping a cluster nobody can reach. With no implicit creator
    # admin and no explicit admins, kubectl is locked out from the start.
    precondition {
      condition     = var.bootstrap_creator_admin || length(var.admin_principal_arns) > 0
      error_message = "bootstrap_creator_admin is false, so list at least one identity in admin_principal_arns or nobody can use the cluster."
    }
  }

  # The cluster policy must be attached before EKS tries to create ENIs.
  depends_on = [aws_iam_role_policy_attachment.cluster]
}

###############################################################################
# Node IAM role. Attached to the EC2 instances via the node group. Every pod
# that uses the node credentials runs as this role.
###############################################################################
data "aws_iam_policy_document" "node_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "node" {
  name               = "${var.cluster_name}-node"
  assume_role_policy = data.aws_iam_policy_document.node_assume.json
  tags               = var.tags
}

# The bootstrap minimum: register with the cluster, pull from ECR, run the CNI.
# AmazonEBSCSIDriverPolicy is the honest shortcut: it belongs on an IRSA role,
# parked here so the EBS addon works until the IRSA session moves it off.
resource "aws_iam_role_policy_attachment" "node" {
  for_each = toset([
    "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
    "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
    "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
    "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy",
  ])

  role       = aws_iam_role.node.name
  policy_arn = each.value
}

###############################################################################
# Bootstrap managed node group. Small, on-demand, three AZs. Hosts the system
# pods plus Karpenter and ArgoCD later. Karpenter owns everything that scales.
###############################################################################
resource "aws_eks_node_group" "bootstrap" {
  cluster_name    = aws_eks_cluster.this.name
  node_group_name = "bootstrap"
  node_role_arn   = aws_iam_role.node.arn
  subnet_ids      = var.subnet_ids

  ami_type       = "AL2023_x86_64_STANDARD"
  capacity_type  = "ON_DEMAND"
  instance_types = var.node_instance_types

  scaling_config {
    desired_size = var.node_desired_size
    min_size     = var.node_min_size
    max_size     = var.node_max_size
  }

  update_config {
    max_unavailable = 1
  }

  tags = var.tags

  depends_on = [aws_iam_role_policy_attachment.node]

  # No ignore_changes on desired_size here on purpose. This group is static and
  # Karpenter does not touch it, so Terraform owns all three numbers. Add
  # ignore_changes = [scaling_config[0].desired_size] only when a cluster
  # autoscaler manages the group. The trade then: you cannot raise min_size
  # past the stale desired_size without an InvalidParameterException.
}

###############################################################################
# Access entries. Explicit cluster-admin for each human ARN you pass in, so a
# person can always reach the cluster even when CI is the creator.
###############################################################################
resource "aws_eks_access_entry" "admin" {
  for_each = toset(var.admin_principal_arns)

  cluster_name  = aws_eks_cluster.this.name
  principal_arn = each.value
  type          = "STANDARD"
}

resource "aws_eks_access_policy_association" "admin" {
  for_each = toset(var.admin_principal_arns)

  cluster_name  = aws_eks_cluster.this.name
  principal_arn = each.value
  policy_arn    = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy"

  access_scope {
    type = "cluster"
  }

  depends_on = [aws_eks_access_entry.admin]
}

###############################################################################
# The four core addons, EKS-managed. kube-proxy and vpc-cni come up with the
# nodes. coredns and the EBS driver need a node to schedule on, hence depends_on.
###############################################################################
resource "aws_eks_addon" "kube_proxy" {
  cluster_name                = aws_eks_cluster.this.name
  addon_name                  = "kube-proxy"
  addon_version               = lookup(var.addon_versions, "kube-proxy", null)
  resolve_conflicts_on_create = "OVERWRITE"
  resolve_conflicts_on_update = "OVERWRITE"
  tags                        = var.tags
}

resource "aws_eks_addon" "vpc_cni" {
  cluster_name                = aws_eks_cluster.this.name
  addon_name                  = "vpc-cni"
  addon_version               = lookup(var.addon_versions, "vpc-cni", null)
  resolve_conflicts_on_create = "OVERWRITE"
  resolve_conflicts_on_update = "OVERWRITE"
  tags                        = var.tags
}

resource "aws_eks_addon" "coredns" {
  cluster_name                = aws_eks_cluster.this.name
  addon_name                  = "coredns"
  addon_version               = lookup(var.addon_versions, "coredns", null)
  resolve_conflicts_on_create = "OVERWRITE"
  resolve_conflicts_on_update = "OVERWRITE"
  tags                        = var.tags

  depends_on = [aws_eks_node_group.bootstrap]
}

resource "aws_eks_addon" "ebs_csi" {
  cluster_name                = aws_eks_cluster.this.name
  addon_name                  = "aws-ebs-csi-driver"
  addon_version               = lookup(var.addon_versions, "aws-ebs-csi-driver", null)
  resolve_conflicts_on_create = "OVERWRITE"
  resolve_conflicts_on_update = "OVERWRITE"
  tags                        = var.tags

  # Bootstrap permissions come from the node role for now. The identity session
  # moves this to a dedicated role, either a Pod Identity association on the addon
  # (the current default, no OIDC provider needed) or IRSA.
  depends_on = [aws_eks_node_group.bootstrap]
}
