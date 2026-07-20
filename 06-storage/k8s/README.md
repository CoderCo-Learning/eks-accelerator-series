# EP6 Kubernetes manifests

Apply these after `terraform apply` in `../terraform/envs/dev`. Edit the role ARN in `irsa-demo.yaml` first (`terraform output demo_role_arn`).

## Storage classes

```bash
kubectl apply -f storageclass-gp3.yaml
kubectl apply -f storageclass-gp3-retain.yaml

# EKS ships a gp2 default. Take the default flag off it so gp3 is the only default.
kubectl patch storageclass gp2 \
  -p '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"false"}}}'

kubectl get sc     # gp3 (default), gp3-retain, gp2 (no default marker)
```

## Snapshot controller (install once)

EKS does not ship the CSI snapshot controller. Install it from the `external-snapshotter` project. Pin a v8 tag and check the releases page for the latest patch.

```bash
SNAP=v8.6.0   # current at time of writing, check the releases page
BASE=https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/$SNAP

kubectl apply -f $BASE/client/config/crd/snapshot.storage.k8s.io_volumesnapshotclasses.yaml
kubectl apply -f $BASE/client/config/crd/snapshot.storage.k8s.io_volumesnapshots.yaml
kubectl apply -f $BASE/client/config/crd/snapshot.storage.k8s.io_volumesnapshotcontents.yaml
kubectl apply -k "https://github.com/kubernetes-csi/external-snapshotter/deploy/kubernetes/snapshot-controller?ref=$SNAP"

kubectl apply -f volumesnapshotclass.yaml
```

## The snapshot and restore drill

```bash
kubectl apply -f demo-pvc-pod.yaml
kubectl exec demo-writer -- cat /data/hello.txt     # hello-from-ep6

kubectl apply -f volumesnapshot.yaml
kubectl get volumesnapshot demo-snap -w             # READYTOUSE true

kubectl apply -f restore-pvc.yaml
kubectl exec demo-reader -- cat /data/hello.txt     # hello-from-ep6, restored
```
