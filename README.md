# EKS Accelerator Series

The sequel to the [ECS Accelerator Series](https://github.com/CoderCo-Learning/ecs-accelerator-series). Built to help CoderCo students through the EKS v2 project.

Each session is a single topic deep dive. The brief for the project is in [project.md](project.md).

## Sessions

- [EP 1: Foundations & Local Dev](01-foundations/README.md)

## Platform snapshot

A snapshot of the EKS v2 application code lives in [`platform/`](platform/README.md) so the series is self contained. Run it locally with `docker compose` before showing up to a session. The upstream source of truth is [CoderCo-Learning/eks-v2](https://github.com/CoderCo-Learning/eks-v2).

## How to use this repo

Each session has its own folder. The folder contains:

- A `README.md` you read before the live session
- Any reference manifests, Terraform snippets or scripts the session walks through

Read the session README before showing up. Carry the work into your EKS v2 project repo.

## What this series is not

- A copy paste solution for the EKS v2 project
- A tutorial you can complete without reading the AWS or Kubernetes docs
- A guarantee that you will pass the live review

You still need to build the project, write the README and defend every resource you provision.
