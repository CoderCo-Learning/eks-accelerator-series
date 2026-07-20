###############################################################################
# OIDC provider. Registers the cluster in IAM as a signer of identity tokens,
# so AWS is willing to trust tokens the cluster issues to service accounts.
# One per cluster. This is the foundation every IRSA role stands on.
###############################################################################
data "tls_certificate" "oidc" {
  url = var.oidc_issuer_url
}

resource "aws_iam_openid_connect_provider" "this" {
  url             = var.oidc_issuer_url
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [data.tls_certificate.oidc.certificates[0].sha1_fingerprint]
  tags            = var.tags
}

locals {
  # The issuer without the scheme, used as the condition key prefix.
  oidc_host = replace(var.oidc_issuer_url, "https://", "")
}

###############################################################################
# EBS CSI driver role. Trusted only by the ebs-csi-controller-sa service
# account, proven through the OIDC provider. This replaces the EP4 node-role
# shortcut. Wire it onto the EP4 addon with service_account_role_arn.
###############################################################################
data "aws_iam_policy_document" "ebs_csi_trust" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.this.arn]
    }

    # Only this exact service account may assume the role.
    condition {
      test     = "StringEquals"
      variable = "${local.oidc_host}:sub"
      values   = ["system:serviceaccount:${var.namespace}:${var.ebs_csi_service_account}"]
    }

    # The token must be meant for STS.
    condition {
      test     = "StringEquals"
      variable = "${local.oidc_host}:aud"
      values   = ["sts.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "ebs_csi" {
  name               = "${var.cluster_name}-ebs-csi"
  assume_role_policy = data.aws_iam_policy_document.ebs_csi_trust.json
  tags               = var.tags
}

resource "aws_iam_role_policy_attachment" "ebs_csi" {
  role       = aws_iam_role.ebs_csi.name
  policy_arn = var.ebs_csi_policy_arn
}

###############################################################################
# Demo role for the deep dive. Same IRSA shape, trusted by a throwaway service
# account. A pod assumes it and runs `aws sts get-caller-identity`, which needs
# no permissions, so no policy is attached. Change demo_service_account to a
# wrong name and re-apply to break the trust and read the error.
###############################################################################
data "aws_iam_policy_document" "demo_trust" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.this.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_host}:sub"
      values   = ["system:serviceaccount:${var.namespace}:${var.demo_service_account}"]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_host}:aud"
      values   = ["sts.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "demo" {
  name               = "${var.cluster_name}-irsa-demo"
  assume_role_policy = data.aws_iam_policy_document.demo_trust.json
  tags               = var.tags
}
