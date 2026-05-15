# Episode 1: Foundations & Local Dev

## Why this episode exists

You just shipped ECS. The next jump is not "ECS but with `kubectl`". The actual jump is running stateful workloads on Kubernetes with the operational story around them.

This episode is the orientation. Read the project brief. Get it running on your laptop with `docker compose`. Drive Kubernetes primitives on a local kind cluster. Then we go to EKS.

If you skip this episode you will spend cluster time debugging the application instead of the cluster. The cluster bill ticks every minute. Save it for the parts that need it.

## What you walk out with

- A mental map of the nine services and how they talk
- The order flow running locally via `docker compose`
- A local `kind` cluster with Pod, Deployment, Service and StatefulSet primitives running against real PVCs
- A reading list for the Go code so you can come into Episode 2 already knowing where state lives

## The project in one paragraph

Nine Go services. One Postgres. One Redis. One SQS queue with a DLQ. A worker that consumes events. A dashboard that drives the UI. You build the infrastructure, the manifests, the pipelines and the observability around it. Postgres and Redis run in the cluster on PVCs. This is the interesting part of the project. The applications are deliberately rough so you focus on the platform, not the code.

Full brief: [project.md](../project.md).

---

## 1. Where Kubernetes fits

You already know:

- **Docker** runs the process
- **ECS** runs the container with AWS managing the scheduler

Kubernetes answers the same question as ECS, with more primitives and more rope.

| ECS thing | Kubernetes thing | Notes |
|---|---|---|
| Task | Pod | A pod can hold more than one container. Most don't |
| Task definition | PodSpec inside a Deployment / StatefulSet | Image, env, ports, resources, probes |
| Service | Deployment + Service | Deployment manages replicas. Service gives them a stable name |
| Service (Fargate) | Pod scheduled by the scheduler onto a node | EKS still has a scheduler, you can see it |
| ALB + target group | Ingress + Ingress Controller (Traefik) | Or Gateway API. Same idea, more layers |
| Task IAM role | ServiceAccount + IRSA | IAM trust is bound to a service account, not a task |
| Secrets in SSM | Secrets Manager via External Secrets | Secrets live in Secrets Manager, sync into K8s Secrets |
| Auto Scaling | HPA (and Karpenter for nodes) | Two scalers. Pods scale on metrics. Nodes scale on pending pods |
| CloudWatch Logs | Cluster logging stack | Promtail or Fluent Bit shipping to Loki or CloudWatch |

New shapes you did not have in ECS:

- **Namespace.** A soft tenant boundary. Not a security boundary on its own
- **StatefulSet.** A Deployment with stable identity and stable storage per pod
- **PersistentVolumeClaim.** A request for a disk. The cluster gives you one or fails to schedule
- **CustomResourceDefinition.** New API kinds. ArgoCD's `Application`, External Secrets' `ExternalSecret`, Karpenter's `NodePool` are all CRDs
- **Controllers.** Reconciliation loops that drive real state towards desired state. ArgoCD, Karpenter, cert-manager, External Secrets are all controllers

Read those last two more than once. Most of EKS work is configuring controllers.

---

## 2. Why we run Postgres and Redis in cluster

The project rules say RDS and ElastiCache are off the table. There is a reason.

Most teams put state on managed services because state on Kubernetes is the hard part. You get to do the hard part on purpose. By the end of the project you will know:

- Why a StatefulSet keeps `postgres-0` bound to the same PVC forever
- What happens to a PVC when the pod's AZ goes down
- How to take a snapshot and restore into a new PVC
- How a single pod restart looks to nine services that all hold open database connections

This is the bit that pays off in a real on call rotation. ECS hides most of it.

---

## 3. The nine services

Read the source before the next session. Code is in `services/` in the EKS v2 project repo.

| Service | Reads | Writes | Talks to |
|---|---|---|---|
| api-gateway | nothing | nothing (auth state in Redis) | order, inventory, payment, shipping, notification, dashboard |
| order-service | Postgres `orders` | Postgres `orders` | inventory, payment, SQS |
| inventory-service | Postgres `inventory` | Postgres `inventory` | Redis (locks) |
| payment-service | Postgres `payments` | Postgres `payments` | external gateway, SQS |
| notification-service | nothing | nothing | SMTP, SMS provider |
| shipping-service | Postgres `shipments` | Postgres `shipments` | carrier API, SQS |
| worker | SQS messages | Postgres (cross service) | order, payment, shipping, notification |
| scheduler | Postgres | Postgres + SQS | order, inventory, payment |
| dashboard-api | Postgres (read mostly) | Postgres | all of the above |

Things to look for while reading:

- Where does each service read its config from? (env vars vs files vs Secrets Manager)
- What port does it expose? Does it expose a `/health` and a `/metrics`?
- Where is the SQS publishing code? Which services emit events?
- Where does the worker decide an event has failed three times?
- Which service owns each table? Does anyone read another service's table?

That last question is the one that determines how you handle migrations later.

---

## 4. Local dev

Run the platform on your laptop before touching AWS.

### Prerequisites

```bash
docker --version          # 24.x or above
docker compose version    # v2
kubectl version --client  # 1.33 or above
kind --version            # 0.24.x or above
go version                # 1.22 or above, if you want to run a service outside compose
```

### Run the project locally

```bash
# in the EKS v2 project repo, not this one
docker compose up --build
```

What this brings up:

- All nine services
- One Postgres
- One Redis
- LocalStack for SQS (or a stub, check the compose file)

### Smoke test

```bash
# api-gateway is the public entrypoint
curl http://localhost:8080/health
# {"status":"ok"}

# place an order
curl -X POST http://localhost:8080/orders \
  -H 'content-type: application/json' \
  -d '{"sku":"abc-123","qty":1,"customer":"test@example.com"}'

# watch the worker logs in another terminal
docker compose logs -f worker
```

You should see the order land, inventory reserve, payment process and a shipping record appear. If any of that is broken, fix it locally before going further. Broken locally means broken on EKS, just slower to debug.

### Map the call graph

Open the compose logs in nine terminals (or one with `--tail=0 -f`). Place an order. Note the order of log lines across services. That is your call graph. Draw it. Put the sketch in your project README on day one. You will refine it later, but you want a baseline.

---

## 5. Lab: a kind cluster and four primitives

Goal: see Pod, Deployment, Service and StatefulSet behave with your own eyes on a local cluster. No EKS yet.

The manifests are in [`lab/`](lab). They are intentionally tiny. The point is the behaviour, not the YAML.

```bash
cd 01-foundations/lab
kind create cluster --config kind-config.yaml --name eks-accel

kubectl get nodes
# NAME                       STATUS   ROLES           AGE   VERSION
# eks-accel-control-plane    Ready    control-plane
# eks-accel-worker
# eks-accel-worker2
```

### Run through the lab steps

The lab README walks each one. Short version:

```bash
# 1. A bare Pod (you almost never run these in real life)
kubectl apply -f 01-pod.yaml
kubectl get pods
kubectl delete pod nginx                  # gone, not coming back

# 2. A Deployment (replicas + self healing)
kubectl apply -f 02-deployment.yaml
kubectl get pods
kubectl delete pod -l app=nginx           # watch it come back

# 3. A Service in front of the Deployment
kubectl apply -f 03-service.yaml
kubectl port-forward svc/nginx 8080:80
curl localhost:8080

# 4. A StatefulSet with a PVC per pod
kubectl apply -f 04-statefulset.yaml
kubectl get pvc
kubectl exec -it data-0 -- sh -c 'echo hi > /data/hello; cat /data/hello'
kubectl delete pod data-0                 # comes back with the same PVC, same file
kubectl exec -it data-0 -- cat /data/hello
```

The point of step 4 is the moment the file survives a pod delete. Sit with that for a minute. That is the whole reason StatefulSets exist.

### Teardown

```bash
kind delete cluster --name eks-accel
```

Don't skip this. `kind` is local but it eats RAM.

---

## 6. Common confusions to flush before next week

- **A Pod is not a unit of scale.** A Deployment is. You scale Deployments, not Pods directly
- **A Service is not a load balancer.** It is a stable DNS name with iptables magic behind it. The load balancer is the Ingress (or Service of type LoadBalancer, which you mostly will not use)
- **`kubectl apply` is declarative, not imperative.** Re running it does not re create. It diffs and converges
- **The control plane runs the cluster. The data plane runs your work.** EKS manages the control plane. Karpenter manages your data plane

If any of those still feel hand wavy, that is the work for this week.

---

## Homework

1. Run the project locally end to end. Place five orders. Read the worker logs
2. Draw the call graph. Push it as `docs/architecture-v0.png` in your project repo
3. Read at least three of the Go services in full. Pick `order-service`, `worker` and one of your choice
4. Finish the kind lab. Delete a `postgres` style pod from the StatefulSet. Confirm the data survives
5. Write a paragraph in your project README about why this project keeps Postgres in cluster. Use your own words

Bring questions to the next session. Episode 2 is containerisation. We write nine Dockerfiles, stand up ECR in Terraform, wire image scanning into the build before anything reaches a cluster.

---

## What's next

[EP 2: Containerisation & ECR](../02-containerisation-ecr/README.md)

Multi stage Dockerfiles for Go services. One ECR repo per service in Terraform. Tagging by commit SHA. Scanning images before they ever reach a cluster.
