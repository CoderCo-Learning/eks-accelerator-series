# Episode 2: The nine services and their containers

## Why this episode

EP1 was about running the platform. EP3 onwards is about putting it on EKS. EP2 sits in the middle — a tour of the nine services so we know what each one is before we hand it to a cluster.

We are not rewriting any Dockerfile today. Every service ships with one already. They are deliberately rough so we can poke at them. What we want is a clear picture of:

- What each service does in one sentence
- What it talks to (Postgres, Redis, SQS, other services)
- What is in its Dockerfile and where it differs from the rest
- The one or two things in that Dockerfile that bite us later in the series

By the end you should be able to point at any of the nine services and say "this one becomes a Deployment, this one a StatefulSet client, this one wants IRSA for SQS, this one is the only Pod that needs Ingress". That sentence is what session 3 builds the network for.

## How this session runs (45 min)

| Block | Mins | What we do |
|---|---|---|
| 0 | 3 | Why a tour, what every service has in common |
| 1 | 27 | Walk the nine services, ~3 min each |
| 2 | 10 | What we just saw. The service-to-session map |
| 3 | 5 | Homework, what next week looks like |

---

## 0. The baseline they all share

Every service in `platform/services/` ships an identical Dockerfile:

```dockerfile
# Lab quality Dockerfile. You will rewrite this in EP2 (multi stage, distroless, non root, .dockerignore, scanned).
FROM golang:1.26-alpine
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o /app/service .
CMD ["/app/service"]
```

It works. It is also lab grade — the comment says so. We tighten it in session 9 when we add probes, resource limits and a proper `securityContext`. Today we leave it alone and focus on what is inside the binary, because that is the bit each service does differently.

The other things every service has in common — worth knowing before we tour:

- `/livez` is a 200 with no body. The process is up
- `/healthz` is a JSON status. The process is up **and** its dependencies (Postgres, Redis) are reachable
- All config comes from environment variables. There are no config files
- All of them handle `SIGTERM` and run a 30s graceful shutdown
- All of them log to stdout. No file logging anywhere

Hold those four points in your head. Each one of them maps to an EKS feature later — probes, ConfigMaps + ExternalSecrets, Pod terminationGracePeriodSeconds, the cluster log shipper.

Now the tour.

---

## 1. api-gateway (port 8080)

**What it does.** The single front door. Auth (login, register, JWT signing), rate limiting, reverse proxy to the other services.

**What it talks to.**
- Redis (rate limiting state, optional — it degrades gracefully if Redis is unreachable)
- Every other HTTP service (via env vars `ORDER_SERVICE_URL`, `INVENTORY_SERVICE_URL` etc)

**Dockerfile note.** Identical to the rest. The binary is small. Vendor deps include `go-redis` and `golang-jwt`. No special build flags needed.

**What bites in EKS.**
- This is **the only Pod the public internet ever touches**. Session 10 puts Traefik + an NLB + cert-manager in front of it
- The `JWT_SECRET` env var is currently `local-dev-secret` from `docker-compose.yml`. In EKS it has to come from Secrets Manager via External Secrets (session 8). If it leaks, every user session in the system is forgeable
- `REDIS_URL` will become `redis://redis-0.redis.default.svc:6379` once Redis is on a StatefulSet (session 7). Stable DNS, not a load balancer
- This Pod is the obvious candidate for HPA on CPU. Session 9. Everything else scales for different reasons.

---

## 2. order-service (port 8081)

**What it does.** Owns the `orders` table. State machine for order lifecycle (pending → confirmed → processing → shipped → delivered). Publishes `order.created` and `order.status_changed` events.

**What it talks to.**
- Postgres (its own `orders` and `order_events` tables)
- SQS via `SQS_QUEUE_URL` (currently LocalStack)
- No direct calls to other services. Communication is via the event bus

**Dockerfile note.** Identical. The interesting bit is what runs at startup, not what builds.

**What bites in EKS.**
- **Runs migrations on startup.** Look at `main.go` — `migrate()` is called before the HTTP server starts. With one replica this is fine. With three replicas starting cold, all three race for the same `CREATE TABLE IF NOT EXISTS`. Postgres handles it but it is sloppy. Session 15 moves migrations into a Kubernetes `Job` that runs once per release
- `db.SetMaxOpenConns(25)` — every replica holds up to 25 connections. Three replicas times nine services that do this = 675 connections to a single Postgres pod. Session 7 caps it
- `publishEvent` will need IRSA with `sqs:SendMessage` scoped to one specific queue ARN. Session 11
- The order state transitions are in-memory only (`validTransitions` map). No cache, no leadership. Safe to scale horizontally.

---

## 3. inventory-service (port 8082)

**What it does.** Stock levels per SKU. Reservations with TTLs. The bit that prevents double-selling the last item.

**What it talks to.**
- Postgres (`products` and `reservations` tables)
- Nothing else. It is called by `order-service` (synchronously) and by the worker (asynchronously)

**Dockerfile note.** Identical.

**What bites in EKS.**
- **The reservation logic is racey.** Two replicas getting two simultaneous `POST /reserve` calls for the last unit could both succeed. In real life this needs a Redis-based distributed lock or a Postgres `SELECT ... FOR UPDATE`. The current code uses the latter loosely. Worth flagging — we will look at the SQL together in session 7
- Same migration-on-startup pattern as `order-service`. Same fix in session 15
- This service is **read-heavy**. In session 14 it is the one we put a cache hit / miss dashboard on first.

---

## 4. payment-service (port 8083)

**What it does.** Charges, refunds, the ledger. Fake processing today (a `math/rand` 90% success rate) but the boundaries are real.

**What it talks to.**
- Postgres (`payments` and `ledger` tables)
- SQS (publishes `payment.processed`, `payment.failed`)
- In real life: a payment provider. We stub it.

**Dockerfile note.** Identical.

**What bites in EKS.**
- This is the **most sensitive Pod in the cluster**. Its IRSA role should grant `sqs:SendMessage` on exactly one queue. Nothing else. Session 11 is where we draw that boundary
- `math/rand` is fine for a demo. In a real system it would never come anywhere near a payment path. Worth saying out loud so nobody copies this pattern home
- Ledger writes need to be transactional with payment status updates. Look at the SQL — they currently are. Session 15 includes a "what happens if you blow up between two writes" exercise targeted at this service.

---

## 5. notification-service (port 8084)

**What it does.** Sends emails and SMS. Stores templates. Logs delivery history. Today it is mocked — every send just inserts a row.

**What it talks to.**
- Postgres (`notifications`, `templates`)
- In real life: SMTP (SES) and SMS (SNS or Twilio). Today: stubs.

**Dockerfile note.** Identical.

**What bites in EKS.**
- This is the service that **needs egress most badly**. In session 3 when we talk about killing the NAT gateway to save money, this is the awkward one — sending email to an external SMTP server needs outbound internet. VPC endpoints help for AWS-native (SES, SNS) but not for a third-party provider
- SMTP creds, SMS provider keys → Secrets Manager, External Secrets. Session 8
- In a real disaster (the email queue backs up overnight), notifications becomes the **noisy neighbour** that hogs Postgres connections. Session 14 alerting catches that.

---

## 6. shipping-service (port 8085)

**What it does.** Creates shipments. Tracks them. Receives webhooks from carriers (`POST /webhook`).

**What it talks to.**
- Postgres (`shipments`, `tracking_events`)
- SQS (publishes shipment status changes)
- In real life: carrier APIs (DHL, UPS, Royal Mail). Stubbed today.

**Dockerfile note.** Identical.

**What bites in EKS.**
- The `/webhook` endpoint is the **second Pod that needs to be reachable from the public internet** — couriers need to POST to it. Session 10 has to route `/webhook/shipping` through Ingress with tighter IP allow-listing than the main app gets
- Webhooks are at-least-once. Same idempotency lesson as the worker. The handler must be safe to call twice with the same payload
- Outbound to carrier APIs hits the same NAT / egress question as notification-service.

---

## 7. scheduler (no main port, health on :8091)

**What it does.** Background cron jobs. Expire abandoned reservations every minute. Detect abandoned carts every 5. Retry failed payments every 15. Generate daily digests every hour. Clean up old events every 30 minutes.

**What it talks to.**
- Postgres only (read-write)
- Indirectly: triggers SQS via the data it writes

**Dockerfile note.** Identical, but the runtime shape is different. Open `main.go` — there is no main HTTP server. Just goroutines on tickers, plus a tiny health server on port 8091.

**What bites in EKS.** This is the most interesting one.

- **You cannot run two of these.** Two scheduler Pods means every cron job fires twice — duplicate reservations expired twice, duplicate digests sent twice. The fix has two shapes:
  - Keep it as a Deployment with **`replicas: 1` and `strategy: Recreate`**. Simple but lossy during deploys
  - **Move each interval job into a Kubernetes `CronJob`**. Cluster-native, each tick is a fresh Pod, no leader election needed. This is what production-grade looks like. Session 9 mentions it; session 15 makes the call
- Health port 8091 is the only thing to probe. There is no app port to expose in the Pod spec
- `db.SetMaxOpenConns(5)` — only 5 connections because nothing else is going through this Pod. Worth knowing for the connection-budget calculation later.

---

## 8. worker (no main port, health on :8090)

**What it does.** Long-polls SQS, fans events out to the other services over HTTP. The async glue of the platform.

**What it talks to.**
- SQS (consumes)
- Every other HTTP service (calls them with events)
- **No Postgres connection**. The only service without one.

**Dockerfile note.** Identical, but again the runtime shape differs. No main HTTP port. Tiny health server on 8090. No database wait at startup.

**What bites in EKS.**
- **HPA on CPU is wrong here.** An idle worker uses no CPU. Bursting it for CPU does nothing useful. The right answer is **KEDA with the SQS scaler** — scale on queue depth. Session 11 introduces it, session 9 mentions where KEDA fits
- SQS delivery is **at-least-once**. Every handler must be idempotent. If you process the same `order.created` event twice, you should not create two orders. The current handlers are roughly idempotent (they call HTTP endpoints that themselves de-dupe) but not airtight. Session 11 talk
- The DLQ is the thing nobody thinks about until messages start landing in it. Session 11 wires up a CloudWatch alarm on DLQ depth so we know
- IRSA for the worker is the **biggest of the nine** by surface area — `sqs:ReceiveMessage`, `sqs:DeleteMessage` and `sqs:GetQueueAttributes` on the main queue, plus the same on the DLQ for inspection. Still scoped to those two queues only.

---

## 9. dashboard-api (port 8086)

**What it does.** Admin UI plus a JSON API for the metrics behind it. Order stats, revenue charts, low-stock alerts, shipping overview.

**What it talks to.**
- Postgres (read-mostly, cross-table queries)
- Indirectly through other services in some endpoints. Mostly direct DB reads for speed.

**Dockerfile note.** Identical on paper. **Different in practice** — look at `main.go`:

```go
//go:embed static
var staticFiles embed.FS
```

The `//go:embed` directive bakes the entire `static/` folder (HTML, CSS, JS) into the compiled Go binary at build time. So even though the Dockerfile looks identical to every other one, this binary is the only one that contains a UI inside it. No separate `COPY static ./static` is needed at runtime. Go does it at compile time.

This is a subtle but important point: **a template Dockerfile works because the language handles the variation for us**. If this UI were a separate `dist/` folder served by a Node process, the Dockerfile would need its own runtime stage with the static files. Because it is Go with embed, it does not.

**What bites in EKS.**
- The UI being baked into the binary means we can rebuild and roll the dashboard with the same SHA-bump pattern as every other service. No separate artefact, no CDN to invalidate
- This is the **third Pod that needs to be reachable from the public internet** — engineers and admins land on it through Ingress. Different ingress path from the customer-facing api-gateway (probably `admin.<domain>` rather than `app.<domain>`), tighter auth
- Read-heavy. Connection pool is `SetMaxOpenConns(10)`. The dashboard queries are big — session 14 includes a "do not bring the cluster down with a 30-second Postgres query from a Grafana panel" cautionary tale, and this service is the example.

---

## What we just saw

Nine services. Nine almost-identical Dockerfiles. Nine very different things going on inside.

Three patterns to lock in:

**By shape.**
| Pattern | Services | What the cluster needs |
|---|---|---|
| HTTP API, talks to Postgres | order, inventory, payment, notification, shipping, dashboard-api | Deployment + Service + Ingress for some |
| HTTP gateway, talks to Redis | api-gateway | Deployment + Service + Ingress + HPA |
| No HTTP API, talks to SQS | worker | Deployment + KEDA + DLQ alarm |
| No HTTP API, ticker-based | scheduler | CronJob (or Deployment with replicas=1) |

**By public reach.**
- api-gateway, shipping-service (webhook), dashboard-api — public internet
- The other six — internal only, never leaves the cluster

**By IRSA need.**
- order, payment, shipping — `sqs:SendMessage` on the events queue
- worker — `sqs:ReceiveMessage`, `sqs:DeleteMessage` on main queue and DLQ
- Everyone else — no AWS API calls at all (currently)

This is the table the rest of the series builds against.

## Service → future session map

| Service | First session it shapes a decision in |
|---|---|
| api-gateway | Session 10 (Ingress, TLS), Session 9 (HPA) |
| order-service | Session 11 (SQS + IRSA), Session 15 (migrations) |
| inventory-service | Session 7 (Postgres locking patterns) |
| payment-service | Session 8 (Secrets), Session 11 (least-privilege IRSA) |
| notification-service | Session 3 (NAT vs VPC endpoints), Session 8 (third-party creds) |
| shipping-service | Session 10 (webhook ingress path) |
| scheduler | Session 9 / 15 (CronJob vs Deployment) |
| worker | Session 11 (SQS, KEDA, DLQ alarms) |
| dashboard-api | Session 10 (admin ingress), Session 14 (heavy reader on the data plane) |

If you read this table backwards, it tells you why we touch the things we touch in the order we touch them.

---

## Homework

1. Open each `main.go` in `platform/services/`. For each one, write down the list of env vars it reads. There should be between 2 and 8 per service
2. For each service, write a single line: "this becomes a `Deployment` / `StatefulSet` / `CronJob`". Defend the answer for the scheduler and the worker out loud to someone (or write it in your README)
3. Run `docker compose up --build` from `platform/` if you have not lately. Place an order. Watch the worker logs catch the event. The async path you see in compose is the same async path we light up in session 11, just with real SQS instead of LocalStack
4. Pick one service you do not understand yet. Read it in full. Bring one question about it to next week

---

## What we are not doing today

- Rewriting any Dockerfile. The current ones are intentionally rough. We tighten them in session 9 when we know what the cluster wants from them
- Building images for ECR. ECR comes in session 13 when CI/CD is wired in
- Talking about probes, resources, IRSA in any depth. Each gets its own session

If you find yourself wanting to do any of those today, hold the thought. Write it down. We will get there.

---

## Next week

Session 3. VPC and network design. Now that we know which Pods need public reach (3 of 9) and which need outbound internet (notification-service in particular), we can design the network around it instead of waving our hands at it.
