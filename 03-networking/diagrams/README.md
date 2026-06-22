# EP3 diagrams

Editable source for the Episode 3 teaching diagram. Open `ep3-networking-architecture.drawio`
in [draw.io](https://app.diagrams.net) (or the VS Code Draw.io Integration extension) and export to PNG/SVG for slides.

The file has three pages (tabs along the bottom):

1. **VPC network architecture** - the full picture. AWS Cloud and the EKS control plane (AWS-managed, outside your VPC), the VPC at `10.0.0.0/16`, the IGW and NLB, three AZs each with a small `/24` public subnet and a large `/19` private subnet, NAT in `single` mode, the S3 gateway endpoint and per-AZ interface-endpoint ENIs, plus the subnet discovery tags and route-table targets. Green lines are ingress, orange dashed is the EKS control plane reaching in via ENIs, purple is private AWS-API traffic via endpoints. Egress to the internet is flagged on the NAT here and broken down in full on page 2, kept off page 1 so the ingress paths stay readable.
2. **Egress: NAT vs VPC endpoints** - the cost lever. The two roads out of a private subnet, the `nat_mode` options (`none` / `single` / `per_az`) with their costs and trade-offs, and the defensible middle answer for this project.
3. **Security group layers** - cluster, node, pod and the endpoints SG, what each attaches to and when you reach for it.

Everything on the pages maps directly to `../README.md` and the reference module in `../terraform/modules/vpc`.
