# EP5 diagrams

Editable source for the Episode 5 teaching diagram. Open `ep5-karpenter-flow.drawio`
in [draw.io](https://app.diagrams.net) (or the VS Code Draw.io Integration extension) and export to PNG/SVG for slides.

Three pages (tabs along the bottom):

1. **Provisioning loop** - a pending pod reaches the controller, the NodePool and EC2NodeClass shape the decision, `CreateFleet` brings up a Spot or on-demand node, the pod schedules. The interruption queue feeds back in. The note captures the 40 to 60 second speed.
2. **Identity and discovery** - the `karpenter` service account to the Pod Identity association to the controller role, left to right. The two footers cover discovery (the `karpenter.sh/discovery` tags on subnets and the cluster SG) and the join (reusing the EP4 node role so its `EC2_LINUX` access entry applies).
3. **Disruption lifecycle** - the four ways a Karpenter node leaves: consolidation, drift and expiry (Karpenter acting) plus interruption (AWS reclaiming). The footer names the brakes: the NodePool budget, PodDisruptionBudgets and the do-not-disrupt annotation.

Colour key: blue is Karpenter and its objects, green is the node and its IAM, amber is the AWS-side plumbing (queue, interruption, discovery), red is the unschedulable pod.
