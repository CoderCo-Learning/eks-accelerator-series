# Platform Snapshot

A snapshot of the application code from the [EKS v2 project repo](https://github.com/CoderCo-Learning/eks-v2) so the series is self contained.

This is what you are building the infrastructure around. Read the Go source, run it locally, then move on to the next session.

> This is a snapshot, not the source of truth. The source of truth lives in `CoderCo-Learning/eks-v2`. Submit against that repo when you grade.

## Run locally

```bash
cd platform
docker compose up --build
```

Smoke test:

```bash
curl http://localhost:8080/livez     # HTTP 200
curl http://localhost:8080/healthz   # {"service":"api-gateway","status":"ok"}
```

The api-gateway is the public entrypoint. The other services listen on `8081` to `8086`. The worker and scheduler expose an internal health port (`8090`, `8091`) but compose does not publish those.

## Services

| Service | Port | DB | SQS | Notes |
|---|---|---|---|---|
| api-gateway | 8080 | no | no | Auth and routing |
| order-service | 8081 | yes | publish | Order lifecycle |
| inventory-service | 8082 | yes | no | Stock and reservations |
| payment-service | 8083 | yes | publish | Payments and ledger |
| notification-service | 8084 | yes | no | Email and SMS |
| shipping-service | 8085 | yes | publish | Shipments and tracking |
| dashboard-api | 8086 | yes | no | Admin UI |
| worker | none | no | consume | SQS consumer |
| scheduler | none | yes | publish | Cron jobs |

## Read the source

Start with `services/order-service/main.go` and `services/worker/main.go`. Those two cover the full event lifecycle.

## What this snapshot is not

- The grading target. Submit against the upstream `eks-v2` repo
- Where you commit your infrastructure work. Build that in a separate repo per the project brief
