# Lab: StatefulSets on a local Kind cluster

You can practise the whole StatefulSet half of this episode on your laptop. Kind gives you a multi-node cluster and a default storage class, so stable identity, per-pod disks, headless DNS and data surviving a restart all work locally. You can even reproduce the AZ-pinning failure, using a node as the stand-in for an Availability Zone.

## What you will do

- Watch `postgres-0` come up with its own disk and a stable DNS name.
- Write a row, kill the pod, then find the row still there.
- See each pod get its own PVC, named after it.
- Reproduce the AZ-pinning failure by trapping a pod away from its disk.

## Prerequisites

```bash
docker --version     # OrbStack or Docker Desktop
kind --version
kubectl version --client
```

## Create the cluster and label fake zones

```bash
cd 07-stateful/lab
kind create cluster --config kind-config.yaml

# label the two workers as pretend Availability Zones
kubectl label node stateful-lab-worker  topology.kubernetes.io/zone=lab-a
kubectl label node stateful-lab-worker2 topology.kubernetes.io/zone=lab-b
kubectl get nodes -L topology.kubernetes.io/zone
```

The zone labels are cosmetic here. The real failure domain on Kind is the node, because a local-path volume is tied to the node it was created on, exactly the way an EBS volume is tied to its AZ.

## 1. Identity, a disk, a name

```bash
kubectl apply -f manifests/postgres.yaml
kubectl rollout status statefulset/postgres

kubectl get pods,pvc -o wide
# pod/postgres-0 on one of the workers
# pvc/data-postgres-0 Bound  <- its own disk, named after it
```

Prove the stable DNS name from another pod:

```bash
kubectl run tmp --rm -it --image=busybox --restart=Never -- \
  nslookup postgres-0.postgres.default.svc.cluster.local
# resolves to postgres-0's IP
```

That name is fixed. Wherever the pod runs, `postgres-0.postgres` always finds it.

## 2. The data outlives the pod

```bash
kubectl exec -it postgres-0 -- psql -U app -d app -c "create table t(x int); insert into t values (42);"
kubectl delete pod postgres-0
kubectl rollout status statefulset/postgres
kubectl exec -it postgres-0 -- psql -U app -d app -c "select * from t;"
# 42. same name, same disk, data intact.
```

## 3. Each pod gets its own disk

Scale up briefly to see the pattern (these become two independent Postgres pods rather than a cluster; we are only showing identity and disks):

```bash
kubectl scale statefulset/postgres --replicas=2
kubectl get pods,pvc
# postgres-0 and postgres-1, plus data-postgres-0 and data-postgres-1
kubectl scale statefulset/postgres --replicas=1
```

`postgres-1` came up after `postgres-0`, with its own fresh PVC. That is the volumeClaimTemplate and the ordering.

## 4. The AZ-pinning failure, locally

Find the node `postgres-0` is on, cordon it, then force the pod to move:

```bash
NODE=$(kubectl get pod postgres-0 -o jsonpath='{.spec.nodeName}')
echo "postgres-0 is on $NODE"
kubectl cordon "$NODE"
kubectl delete pod postgres-0

kubectl get pod postgres-0            # Pending
kubectl describe pod postgres-0 | tail
# ... didn't match Pod's node affinity/selector: the volume is on the cordoned node
```

The pod cannot follow its disk to another node, the same way it cannot follow an EBS volume across AZs. Recover it:

```bash
kubectl uncordon "$NODE"
kubectl rollout status statefulset/postgres
```

## 5. Valkey, the same pattern

```bash
kubectl apply -f manifests/valkey.yaml
kubectl rollout status statefulset/valkey
kubectl exec -it valkey-0 -- valkey-cli set hello world
kubectl delete pod valkey-0
kubectl rollout status statefulset/valkey
kubectl exec -it valkey-0 -- valkey-cli get hello     # "world", from the append-only file
```

## How this maps to EKS

| In this lab (Kind) | On EKS (this episode) |
|---|---|
| local-path volume, tied to a node | EBS volume, tied to an AZ |
| Cordon the node, pod stuck | Node/AZ lost, pod stuck the same way |
| `standard` default class | `gp3` default class |
| Node is the failure domain | AZ is the failure domain |

Same objects, same lesson. The disk is welded to one place, so the pod is too.

## Clean up

```bash
kind delete cluster --name stateful-lab
```
