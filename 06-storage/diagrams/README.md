# EP6 diagrams

Editable source for the Episode 6 teaching diagram. Open `ep6-storage.drawio` in
[draw.io](https://app.diagrams.net) and export to PNG/SVG for slides.

Three pages:

1. **Provisioning flow** - PVC to StorageClass to EBS driver to volume, with the pod triggering the binding. The note explains why binding waits for the pod (AZ).
2. **IRSA trust chain** - service account to signed token to STS to role, with the temporary keys handed back. The footer is the trust policy, the part that says who may assume the role.
3. **Snapshot and restore** - PVC to VolumeSnapshot to VolumeSnapshotContent, then a restore PVC pulling the snapshot back into a fresh volume.

Colour key: blue is the Kubernetes request side, green is the AWS resource and the role, amber is the token and snapshot plumbing, red is the pod that forces AZ selection.
