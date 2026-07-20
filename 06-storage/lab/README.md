# Lab: storage on a local Kind cluster

Storage is the one part of this episode you can practise without an AWS bill. Kind (Kubernetes in Docker) ships a built-in storage class, so you can feel dynamic provisioning, `WaitForFirstConsumer`, access modes and reclaim policy on your laptop. The ideas are identical to EKS. Only the disk underneath changes: a folder on the Kind node here, an EBS volume there.

The one thing you cannot do locally is IRSA, because that is AWS identity. Everything else in the storage half maps one to one.

## What you will do

- Watch a PVC sit `Pending` until a pod needs it, the local version of the AZ trap.
- Write a file to a volume, kill the pod, then find the file still there.
- See `ReadWriteOnce` mean one node at a time.
- Watch `reclaimPolicy` decide whether your data survives deleting the claim.
- Optional: take a real VolumeSnapshot and restore it.

## Prerequisites

```bash
docker --version     # Kind runs nodes as containers
kind --version       # brew install kind (or go install sigs.k8s.io/kind@latest)
kubectl version --client
```

## Create the cluster

```bash
cd 06-storage/lab
kind create cluster --config kind-config.yaml     # one control plane, two workers

kubectl get nodes
# storage-lab-control-plane, storage-lab-worker, storage-lab-worker2   Ready

kubectl get storageclass
# standard (default)   rancher.io/local-path   Delete   WaitForFirstConsumer
```

That default `standard` class is the local mirror of your `gp3` class: dynamic provisioning, `WaitForFirstConsumer` binding, `Delete` reclaim.

## 1. WaitForFirstConsumer: the PVC that waits

Apply a claim on its own and look at it:

```bash
kubectl apply -f manifests/pvc.yaml
kubectl get pvc data
# STATUS: Pending
```

`Pending` here is expected. The class binds `WaitForFirstConsumer`, so it will not create the disk until a pod actually needs it. On EKS this is what stops the volume being made in the wrong Availability Zone. Locally it stops the disk being made on the wrong node.

Now give it a consumer:

```bash
kubectl apply -f manifests/pod.yaml
kubectl get pvc data          # STATUS: Bound
kubectl get pv                # a PV appeared, created for this claim
kubectl get pod writer -o wide   # note which node it landed on
```

The moment the pod scheduled, the class made the disk on that pod's node and bound it. That is dynamic provisioning.

```bash
kubectl exec writer -- cat /data/hello.txt     # hello-from-the-lab
```

## 2. The data outlives the pod

Delete the pod and bring it back:

```bash
kubectl delete pod writer
kubectl apply -f manifests/pod.yaml
kubectl exec writer -- cat /data/hello.txt     # still hello-from-the-lab
```

The container was thrown away and remade. The disk stayed, with the file still on it. That is the whole point of a PersistentVolume.

## 3. ReadWriteOnce means one node at a time

The volume is bound to the node the writer runs on. Try to use it from two pods on different nodes:

```bash
kubectl delete pod writer
kubectl apply -f manifests/readers.yaml
kubectl get pods -o wide
```

Anti-affinity forces the two replicas onto different nodes. One reads the file happily. The other is stuck. Its events say so:

```bash
kubectl describe pod <the-pending-reader> | tail
# ... volume node affinity conflict / cannot use the volume from this node
```

That is `ReadWriteOnce`. The disk belongs to one node at a time. On EKS the reason is the same: an EBS volume attaches to a single server. When you need shared storage across nodes, that is a different service (EFS).

```bash
kubectl delete -f manifests/readers.yaml
```

## 4. reclaimPolicy: does the data survive the claim

The `standard` class is `Delete`. Prove it:

```bash
kubectl delete pvc data       # the PV and the folder behind it go too
kubectl get pv                # gone
```

Now the same story with a `Retain` class:

```bash
kubectl apply -f manifests/retain-storageclass.yaml
# edit manifests/pvc.yaml storageClassName to local-retain (or apply a copy)
```

Delete a claim on `local-retain` and the PV stays in `Released` with the data intact, waiting for you. That is the local mirror of `gp3-retain`, the class you use for a database so a fumbled delete does not wipe it.

## How this maps to EKS

| In this lab (Kind) | On EKS (this episode) |
|---|---|
| `standard` class, `rancher.io/local-path` | `gp3` class, `ebs.csi.aws.com` |
| PVC `Pending` until a pod schedules | same, then it picks the pod's AZ |
| Disk is a folder on the node | an EBS volume in the node's AZ |
| ReadWriteOnce, one node | ReadWriteOnce, EBS single-attach |
| `Delete` vs `Retain` | identical |
| No IRSA | the driver runs on its own IAM role |

Same objects, same behaviour. The cloud version swaps the local folder for an EBS volume and adds the IAM piece.

## Clean up

```bash
kind delete cluster --name storage-lab
```

## Optional: real snapshots, locally

The `standard` class cannot snapshot. To feel the snapshot flow from the episode on your laptop, install the CSI snapshot controller and the CSI hostpath driver, which do support it.

```bash
# 1. the snapshot CRDs and controller (same project as the cloud session)
SNAP=v8.6.0
BASE=https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/$SNAP
kubectl apply -f $BASE/client/config/crd/snapshot.storage.k8s.io_volumesnapshotclasses.yaml
kubectl apply -f $BASE/client/config/crd/snapshot.storage.k8s.io_volumesnapshots.yaml
kubectl apply -f $BASE/client/config/crd/snapshot.storage.k8s.io_volumesnapshotcontents.yaml
kubectl apply -k "https://github.com/kubernetes-csi/external-snapshotter/deploy/kubernetes/snapshot-controller?ref=$SNAP"

# 2. the CSI hostpath driver (a local CSI driver that can snapshot)
git clone https://github.com/kubernetes-csi/csi-driver-host-path.git
cd csi-driver-host-path
./deploy/kubernetes-latest/deploy.sh
```

That gives you a `csi-hostpath-sc` StorageClass and a driver that supports VolumeSnapshots. From there the objects are exactly the ones in `../k8s`: a VolumeSnapshotClass (driver `hostpath.csi.k8s.io`), a VolumeSnapshot of a PVC, then a restore PVC with a `dataSource` pointing at the snapshot. Snapshot a PVC that has a file on it, restore into a new PVC, confirm the file is there.
