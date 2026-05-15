# Lab: kind cluster and four primitives

A local Kubernetes cluster on Docker (`kind`) and four manifests to make Pod, Deployment, Service and StatefulSet behave under your hands.

No EKS yet. The point is to see the primitives respond before you pay for a cluster.

## Prereqs

Docker, `kubectl` and `kind` installed. If you have not done this yet, install them from the [Prerequisites](../README.md#prerequisites) section in EP1 first.

## Create the cluster

```bash
kind create cluster --config kind-config.yaml --name eks-accel
kubectl cluster-info --context kind-eks-accel

# workers take a few seconds after `kind` returns; wait for Ready
kubectl wait --for=condition=Ready node --all --timeout=120s
kubectl get nodes
```

You should see one control plane and two workers. `kind` runs each node as a Docker container.

## Walk through the manifests

### 1. A bare Pod

```bash
kubectl apply -f 01-pod.yaml
kubectl get pods -w        # ctrl-c when Running

# kill it
kubectl delete pod nginx
kubectl get pods           # gone, not coming back
```

Lesson: a bare Pod has no controller. Nothing brings it back.

### 2. A Deployment

```bash
kubectl apply -f 02-deployment.yaml
kubectl get pods -l app=nginx        # three pods

# kill one (pick the first pod name and delete it)
POD=$(kubectl get pods -l app=nginx -o name | head -1)
kubectl delete "$POD"
kubectl get pods -l app=nginx -w     # a new one shows up with a new name
```

Lesson: the Deployment owns a ReplicaSet that owns the Pods. Delete a pod, get a new one. Delete the Deployment, the lot is gone.

### 3. A Service

```bash
kubectl apply -f 03-service.yaml
kubectl get svc nginx

# port forward from your laptop
kubectl port-forward svc/nginx 8080:80
# in another terminal
curl localhost:8080
```

Lesson: a ClusterIP Service is a stable DNS name (`nginx.default.svc.cluster.local`) and a virtual IP. Pods come and go. The Service doesn't.

### 4. A StatefulSet with PVCs

```bash
kubectl apply -f 04-statefulset.yaml
kubectl get pods -l app=data -w      # data-0, data-1 in order
kubectl get pvc                       # one PVC per pod

# write something
kubectl exec data-0 -- sh -c 'echo hi from pod-0 > /data/hello'
kubectl exec data-0 -- cat /data/hello

# delete the pod
kubectl delete pod data-0
kubectl get pods -l app=data -w      # comes back as data-0, same name

# the file is still there
kubectl exec data-0 -- cat /data/hello
```

Lesson: `data-0` is bound to its PVC for life. Deleting the pod does not delete the disk. This is the only reason to use a StatefulSet.

## Things to try if you have time

Run these while the Deployment and StatefulSet are still up. The next section deletes them.

- Scale the Deployment up to 10: `kubectl scale deployment nginx --replicas=10`. Watch the new pods schedule across workers
- Scale the StatefulSet up to 3: `kubectl scale statefulset data --replicas=3`. Watch the pods come up in order (0, 1, 2)
- Scale the StatefulSet back down to 1: `kubectl scale statefulset data --replicas=1`. Watch the pods come down in reverse order (2, 1, 0)
- Run `kubectl describe statefulset data`. Read the events
- Run `kubectl get storageclass`. The default class in `kind` is `standard`, backed by the local path provisioner. On EKS you will swap this for `gp3`

## Now do something destructive on purpose

```bash
kubectl delete statefulset data
kubectl get pvc           # PVCs are still there
kubectl delete pvc --all  # now they go
```

Lesson: deleting a StatefulSet does not delete its PVCs. You delete those explicitly. This bites people when they reinstall a chart and lose their data because they ran `helm uninstall` and then `kubectl delete pvc --all`.

## Teardown

```bash
kind delete cluster --name eks-accel
```
