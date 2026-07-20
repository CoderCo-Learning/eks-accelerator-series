# oidc_issuer_url comes from EP4:
#   cd ../../../../04-cluster/terraform/envs/dev && terraform output oidc_issuer_url
# Once every env is on a remote backend you can read it from state instead of pasting.

module "storage" {
  source = "../../modules/storage"

  cluster_name    = var.cluster_name
  oidc_issuer_url = var.oidc_issuer_url
}
