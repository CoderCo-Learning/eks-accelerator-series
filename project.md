# EKS v2 Project

Nine service order platform on Amazon EKS. Real stateful workloads on Kubernetes. The Go application code is provided. You build the infrastructure, the manifests, the pipelines and the operational story around it.

The full brief lives in the EKS v2 project repo and on Skool. This file is the short version you keep open while working.

## Objective

Deploy a working order platform to EKS, reachable over HTTPS at `app.<your-domain>`, with Postgres and Redis running in cluster on persistent volumes, GitOps driving deploys and a CI pipeline that uses OIDC.

## Services

| Service | Purpose |
|---|---|
| api-gateway | Auth, rate limiting, routes to internal services |
| order-service | Order lifecycle and state machine |
| inventory-service | Stock management and reservations |
| payment-service | Payment processing, refunds, ledger |
| notification-service | Email and SMS dispatch |
| shipping-service | Shipments, tracking, carrier webhooks |
| worker | SQS consumer, orchestrates cross service events |
| scheduler | Cron jobs (expired reservations, abandoned orders, retries) |
| dashboard-api | Admin UI, analytics, reporting |

## Hard requirements

- EKS 1.33 or above across 3 AZs with managed node groups
- Karpenter for node autoscaling
- Postgres on a StatefulSet with a 20Gi gp3 PVC, encrypted
- Redis on a StatefulSet with AOF and a 10Gi gp3 PVC, encrypted
- AWS EBS CSI Driver with IRSA, gp3 default, a VolumeSnapshotClass configured
- SQS queue with a DLQ as the event bus (or in cluster Kafka via Strimzi if you defend the choice)
- ECR repositories, one per service
- VPC with private subnets, no NAT if you can avoid it
- Secrets sourced from AWS Secrets Manager via External Secrets or the Secrets Store CSI Driver
- Traefik as the Ingress controller fronted by an AWS NLB
- cert-manager with Let's Encrypt for TLS
- ExternalDNS managing Route 53 records
- GitHub Actions with OIDC for CI, IRSA for pods
- ArgoCD in cluster, App of Apps pattern, auto sync on dev
- Zero downtime rollouts with rollback on failure
- Least privilege IAM
- Terraform with remote state

## Rules

- Postgres and Redis run in cluster. RDS and ElastiCache are not options
- OIDC for CI. IRSA for pods. No long lived AWS keys anywhere
- If you cannot explain a resource you committed, you did not build it
- Tear down the cluster when not in use. EKS, EBS, NLB and data transfer add up fast

## Grading

- All nine services running and healthy
- End to end flow works through the dashboard UI: create order, reserve inventory, process payment, ship, deliver
- Postgres and Redis on StatefulSets with PVCs that survive pod restarts
- Volume snapshot taken and restored successfully
- HTTPS at a real DNS name
- Pipeline deploys only what changed
- No secrets in Git
- README covers architecture, deployment pipeline, secrets management, storage and restore procedure, scaling strategy, database migration approach
- Live review where you explain every resource you created

## What this series does for you

Walks the project end to end across 15 sessions. Each session covers one topic in depth. By the end you will have built every piece at least once in a learning context, so the project is about putting them together rather than meeting them for the first time.
