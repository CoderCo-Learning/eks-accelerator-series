# EKS Accelerator Series

This series covers the EKS v2 project end to end. Each session is a single topic deep dive.

The project you are building is described in [project.md](project.md).

## Sessions

- [EP 1: Foundations & Local Dev](01-foundations/README.md)
- EP 2: Containerisation & ECR
- EP 3: VPC & Network Design
- EP 4: EKS Cluster with Terraform
- EP 5: Karpenter & Node Autoscaling
- EP 6: Storage, EBS CSI & IRSA
- EP 7: Postgres & Redis on StatefulSets
- EP 8: Secrets, External Secrets & Rotation
- EP 9: Application Manifests, Probes & HPA
- EP 10: Ingress with Traefik, cert-manager & ExternalDNS
- EP 11: Event Bus, Worker & IRSA per Service
- EP 12: GitOps with ArgoCD
- EP 13: CI/CD with GitHub Actions & OIDC
- EP 14: Observability with kube-prometheus-stack
- EP 15: Production Readiness, DR & Chaos

## How to use this repo

Each session has its own folder. The folder contains:

- A `README.md` you read before the live session
- A `lab/` directory when there is hands on work to do
- Any reference manifests, Terraform snippets or scripts the session walks through

Read the session README before showing up. Run the lab during or after the session. Carry the work into your EKS v2 project repo.

## What this series is not

- A copy paste solution for the EKS v2 project
- A tutorial you can complete without reading the AWS or Kubernetes docs
- A guarantee that you will pass the live review

You still need to build the project, write the README and defend every resource you provision.
