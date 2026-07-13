# The values below come from earlier sessions:
#   cluster_name              literal, matches EP4
#   node_role_arn             EP4:  terraform output node_role_arn
#   private_subnet_ids        EP3:  terraform output private_subnet_ids
#   cluster_security_group_id EP4:  terraform output cluster_security_group_id
#
# Once every env is on a remote backend you can read them straight from state
# with terraform_remote_state instead of pasting.

module "karpenter" {
  source = "../../modules/karpenter"

  cluster_name              = var.cluster_name
  node_role_arn             = var.node_role_arn
  private_subnet_ids        = var.private_subnet_ids
  cluster_security_group_id = var.cluster_security_group_id

  karpenter_version = var.karpenter_version
}
