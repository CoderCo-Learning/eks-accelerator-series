# EP4 diagrams

Editable source for the Episode 4 teaching diagram. Open `ep4-cluster-architecture.drawio`
in [draw.io](https://app.diagrams.net) (or the VS Code Draw.io Integration extension) and export to PNG/SVG for slides.

The file has three pages (tabs along the bottom):

1. **Control plane vs data plane** - the split that explains everything. The AWS-managed account on the left (API server, etcd, scheduler), your VPC on the right with the bootstrap nodes in three private subnets, the control-plane ENIs reaching in (dashed amber) and a node calling the API on 443 (green). The endpoint note spells out public-locked plus private.
2. **Three IAM roles** - cluster role, node role, IRSA. Each box names who assumes it, which policies it carries and what job it does. IRSA is dashed and greyed because it is next session. The EBS shortcut on the node role is called out.
3. **Access entry flow** - your ARN to access entry to policy association to working kubectl, left to right. The footer captures the break-it lesson: remove the entry and you get Unauthorized, `terraform apply` brings it back through the EKS API, recoverable because that is AWS IAM and not Kubernetes RBAC.

Everything on the pages maps to `../README.md` and the reference module in `../terraform/modules/eks`.
Colour key: blue is AWS-managed control plane, green is your data plane, amber is the access and reach-in plumbing, grey-dashed is next session.
