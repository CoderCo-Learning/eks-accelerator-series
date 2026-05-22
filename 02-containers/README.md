# Episode 2: Containers (the bit before EKS)

## Why this episode

Last week we ran the platform on `docker compose`. The Dockerfiles were honest about being lab grade — the comment in each one literally says "you will rewrite this in EP2".

Today is EP2. We rewrite them.

This is the last episode before AWS shows up. Once we cross into session 3 we are talking VPCs, subnets, EKS control planes. The images we ship to that cluster need to behave. Today we make sure they do.

## What we walk out with

- A multi-stage Dockerfile we wrote line by line, not copy pasted
- A working `.dockerignore` so we stop shipping `.git` to production
- One service rebuilt the right way live in the session
- A clear hand off to next week: this image becomes a Pod, the SHA we tag becomes the rollback button

## How this session runs (45 min)

| Block | Mins | What we do |
|---|---|---|
| 0 | 2 | Recap of EP1, where this fits |
| 1 | 5 | Read the naive Dockerfile we already have |
| 2 | 5 | Build it. Measure it. Find the smell |
| 3 | 5 | Fix the cache. Rebuild |
| 4 | 12 | Multi-stage + distroless + non-root |
| 5 | 3 | `.dockerignore` |
| 6 | 5 | The other eight services. Templates not clones |
| 7 | 5 | Tag by SHA. Why `:latest` is a footgun in EKS |
| 8 | 3 | Where this lands next week |

Total 45. We will run over by 5. Plan for it.

---

## 0. Recap

Last week we ran the order flow on a laptop. We brought up Postgres, Redis, LocalStack and nine Go services with `docker compose up --build`. The Dockerfiles built every time. They were also wrong in five different ways.

Today we fix that. Same code. Same compose file. Different Dockerfile.

> If you have not run EP1 end to end, do that first. The smoke test in [01-foundations/README.md](../01-foundations/README.md#smoke-test) is the bar. We will assume orders are placing locally.

---

## 1. Read the Dockerfile we already have

Open `platform/services/order-service/Dockerfile` in your editor. All nine services have an identical one. Here it is:

```dockerfile
# Lab quality Dockerfile. You will rewrite this in EP2 (multi stage, distroless, non root, .dockerignore, scanned).
FROM golang:1.26-alpine
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o /app/service .
CMD ["/app/service"]
```

It works. We saw it work last week. Five orders went through this image. So what is wrong with it?

Pause for 60 seconds before reading on. Try to list the problems yourself. We will compare lists.

What we should land on:

1. **The Go compiler is in the runtime image.** A 300MB toolchain that runs once at build time is sitting in our final image, on every node, forever
2. **It runs as root.** Default `USER` in `golang:1.26-alpine` is root. Any container escape gets a root shell on the node
3. **`COPY . .` before `go mod download` kills the layer cache.** Edit one line of Go and the dependency layer rebuilds from scratch
4. **No `.dockerignore`.** `.git`, the local Postgres data dir if we ever copy it, any stray `.env` — all of it lands in the build context
5. **No health signal baked in.** Kubernetes wants a probe. We will fix that properly in session 9 but the image needs to make it possible

Some of these are size problems. Some are security problems. One of them — number 3 — is a developer experience problem that costs us minutes every push. We are going to fix all of them.

---

## 2. Build it and measure

We build the naive image first so we have a baseline to beat.

```bash
cd platform/services/order-service
docker build -t order-service:naive .
docker images order-service
```

Note the size. On my machine right now it is around **440MB**. Anywhere from 380 to 500 is normal.

Now look at where the bytes went:

```bash
docker history order-service:naive
```

Two layers dominate. The base image is one. The other is the `COPY . .` plus the build. The compiler itself is most of that base. We do not need it at runtime.

Sanity check it still runs:

```bash
docker run --rm -e DATABASE_URL=postgres://nope -e PORT=8081 order-service:naive
# expected: connects to nothing, dies. that's fine. it started.
```

We have a baseline. Now we fix it.

---

## 3. Cache the dependencies properly

The fastest win is reordering two lines. We copy `go.mod` and `go.sum` first, run `go mod download`, then copy the rest of the source. Docker caches each layer by the inputs above it. If only the Go source changed, the dependency layer stays cached.

Open `02-containers/lab/Dockerfile.cached`:

```dockerfile
FROM golang:1.26-alpine
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app/service .

CMD ["/app/service"]
```

Build it once. Then touch `main.go` and build again:

```bash
docker build -f ../../02-containers/lab/Dockerfile.cached -t order-service:cached .
touch main.go
docker build -f ../../02-containers/lab/Dockerfile.cached -t order-service:cached .
```

The second build should hit `CACHED` on the `go mod download` layer. That is the win. On a service with a lot of dependencies this is the difference between a 4 second rebuild and a 90 second rebuild.

> Try not to think of this as a Docker trick. It is a property of every layered build system. Bazel, BuildKit, Nix — same idea, different syntax.

The image is still 440MB. We have not made it smaller yet. We have only made it faster to rebuild.

---

## 4. Multi-stage, distroless, non-root

This is the real rewrite. Open `02-containers/lab/Dockerfile`:

```dockerfile
# syntax=docker/dockerfile:1.7

# ---- builder ----
FROM golang:1.26-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/service .

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/service /app/service

USER 65532:65532
EXPOSE 8081
ENTRYPOINT ["/app/service"]
```

We walk it line by line. The questions worth asking out loud:

**Why two `FROM` lines?**
This is what multi-stage means. The builder stage has the compiler. The runtime stage starts fresh and only copies the compiled binary across. Anything in the builder that we do not `COPY --from=builder` is discarded.

**Why `CGO_ENABLED=0`?**
Because distroless has no libc. We compile a fully static binary so we do not depend on a dynamic linker that is not there.

**Why `-trimpath` and `-ldflags="-s -w"`?**
`-trimpath` strips local file paths from the binary — small reproducibility win, also a privacy win. `-s -w` drops the symbol table and DWARF debug info. The binary gets ~25% smaller. We trade off `panic` traces being harder to read. For most services that is the right trade.

**Why `gcr.io/distroless/static-debian12:nonroot`?**
"Distroless" means no shell, no package manager — nothing in the image except what we put there. The `:nonroot` tag ships with a user `nonroot` at UID 65532 already present, which means `USER 65532:65532` works out of the box.

The price you pay: no `docker exec -it container sh` to debug a running pod. There is no shell to exec into. Live with it. The right way to debug a running container in Kubernetes is `kubectl debug` with an ephemeral container — session 14 territory.

**Why `USER 65532:65532`?**
Default container user is root. If anyone breaks out of the container, we do not want them landing as UID 0 on the node. Running as a known non-zero UID also makes it easy to lock down `runAsNonRoot: true` in the Pod spec later, which we will do in session 9.

**Why `EXPOSE 8081`?**
Documentation only. It does not actually open the port — Kubernetes will do that based on the Service spec. But it tells anyone reading the Dockerfile what the contract is.

Now build it:

```bash
docker build -f ../../02-containers/lab/Dockerfile -t order-service:slim .
docker images order-service
```

Expect somewhere around **15 to 20MB**. Down from 440. That is roughly a 95% reduction.

Run it the same way as before:

```bash
docker run --rm -e DATABASE_URL=postgres://nope -e PORT=8081 order-service:slim
```

It dies the same way the naive one did. Good. The behaviour is the same. The image is 25 times smaller. Nothing in the runtime can be exploited via a Go compiler we are not using because the compiler is gone.

> Quick aside: try `docker run --rm -it order-service:slim sh`. It will error with `exec format error` or `no such file`. That is the point of distroless. No shell to land in.

---

## 5. `.dockerignore`

Without one, Docker tarballs the entire build context — everything in the directory you ran `docker build` from — and ships it to the daemon. For our services that means `.git`, any local `vendor/` directory, OS junk, build artefacts. Sometimes secrets we did not mean to copy.

Drop `02-containers/lab/.dockerignore` into each service folder:

```
.git
.gitignore
.dockerignore

# editor
.vscode
.idea
*.swp

# go
vendor/
bin/
*.test
*.out
coverage.out

# os
.DS_Store
Thumbs.db

# env
.env
.env.*
!.env.example

# misc
README.md
*.md
docs/
```

Two notes on that file. We ignore `README.md` because it does not need to be in the image — saves a few KB and avoids accidentally shipping internal notes. We do not ignore `go.mod` or `go.sum` because we need them for `go mod download`.

Rebuild and watch the "Sending build context to Docker daemon" line. On `order-service` it should drop from a few MB to under 200KB.

---

## 6. The other eight services

We are not going to rewrite nine Dockerfiles live. We will rewrite one (we just did) and then talk about the two that are different.

**The worker** — `platform/services/worker/Dockerfile`. This service has no HTTP port. It long-polls SQS, fans out events. So:

- No `EXPOSE` line in the runtime stage
- The Service definition we write later will be headless — there is nothing to load balance
- Liveness will not be `/healthz`; it will be the process being alive at all. Session 9.

Everything else about the Dockerfile is the same as `order-service:slim`.

**The dashboard-api** — `platform/services/dashboard-api/Dockerfile`. This one serves a UI from `/static`. The naive reflex is "I need to COPY the static folder into the runtime stage". Open `main.go`:

```go
//go:embed static
var staticFiles embed.FS
```

The `//go:embed` directive bakes the static folder into the compiled binary at build time. So our runtime stage only needs the binary — same as every other service. The static assets ride along inside it.

This is the lesson: a template works. Cloning does not. Six services share an identical Dockerfile. One drops the `EXPOSE` line. One does not need a separate `COPY` for assets because the language handles it. **Read the code before you copy the Dockerfile.**

The other six (`api-gateway`, `inventory-service`, `payment-service`, `notification-service`, `shipping-service`, `scheduler`) are identical to `order-service` apart from the `EXPOSE` port number. We will template that with a small `Makefile` for homework.

---

## 7. Tag by SHA. Why `:latest` is a footgun

When we push to ECR next week, every tag is a contract. Two patterns to know about:

**Mutable tags** (`:latest`, `:dev`, `:main`) — point to whatever was pushed most recently. Good for local dev. Bad for clusters, because two Pods can pull the "same" tag at different times and get different binaries. Rollback becomes "rebuild the previous code and push it to the same tag", which is slow and lossy.

**Immutable tags** (the Git SHA) — point to one specific build forever. Rollback becomes "edit one line in a manifest to the previous SHA". Fast. Auditable. Survives a 3am page.

The pattern we want:

```bash
SHA=$(git rev-parse --short HEAD)
docker build -t order-service:$SHA -f 02-containers/lab/Dockerfile platform/services/order-service
docker tag order-service:$SHA order-service:latest    # for local convenience
```

Both tags exist. The SHA is the truth. `:latest` is a courtesy.

There is a tiny `Makefile` in `02-containers/lab/` that does this across all nine services. Copy it into your own work. We wire ECR push into it in session 13 when we set up OIDC and GitHub Actions.

> ECR has a setting called **tag immutability**. We will turn it on in session 13. Once on, you cannot overwrite a tag — pushing `order-service:abc123` twice with different content errors. It saves you from a class of bug that only shows up when you really need it not to.

---

## 8. Where this lands next week

Quick preview. Do not implement anything yet.

Next week is VPC. The week after is the EKS cluster. The week after that we install Karpenter so nodes appear when Pods need them. None of that uses the image we built today, directly. But all of it exists so that this command works:

```bash
kubectl run order-service --image=$ACCOUNT.dkr.ecr.eu-west-2.amazonaws.com/order-service:abc123
```

A node has to exist. It has to have the permission to pull from ECR. It has to be on a subnet that can route to ECR endpoints. The Pod has to be allowed to come up as UID 65532 because we told it to.

That is the bridge. Every line we wrote in the Dockerfile today shows up in a cluster decision next month:

- `USER 65532:65532` → `securityContext.runAsNonRoot: true` in the PodSpec
- `EXPOSE 8081` → `containerPort: 8081` in the PodSpec and `targetPort: 8081` in the Service
- Static binary → no surprise glibc mismatch when we move from Alpine to distroless nodes
- Small image → faster Pod startup, faster Karpenter scale up, less ECR data transfer cost

We did not build a Docker image today for the sake of it. We built the unit of deployment for the next 12 weeks.

---

## Lab artefacts

Everything we touched in this session lives under `02-containers/lab/`:

- `Dockerfile` — the multi-stage version we wrote
- `Dockerfile.cached` — the intermediate step (kept for the diff)
- `.dockerignore` — copy this into every service folder
- `Makefile` — `make build` builds all nine, tagged by short SHA

The lab Dockerfiles are deliberately separate from `platform/services/*/Dockerfile`. That way you can diff the naive against the rewrite when revisiting this episode.

---

## Homework

1. Replace each `platform/services/*/Dockerfile` with a multi-stage build. The `worker` and `dashboard-api` need attention — the rest are a copy job
2. Drop the `.dockerignore` into every service folder
3. Run `make build` from `02-containers/lab/`. Expect 9 images, each tagged with the short SHA, each under 25MB
4. Run `docker compose up --build` from `platform/`. Place an order via the smoke test from [EP1](../01-foundations/README.md#smoke-test). The behaviour should be identical to last week with images 20x smaller
5. Push the rewritten Dockerfiles to your fork. Comment on the PR / commit with the before and after image sizes for one service

---

## Common confusions

- **"Distroless still has Debian in the name. Is it Debian?"** Yes. The static-debian12 base inherits a tiny Debian rootfs (CA certs, tzdata, `/etc/passwd` with the `nonroot` user). No package manager, no shell. The "Debian" is the source of those few files, not a running OS
- **"Why not Alpine for everything?"** Alpine uses musl libc, not glibc. CGO code that links against glibc breaks on Alpine. Our Go services build with `CGO_ENABLED=0` so they do not care — but anything you migrate from another language might. Distroless dodges that conversation entirely
- **"What about `scratch`?"** Same idea as distroless but with literally nothing — not even CA certificates. Fine for a Go binary that never makes outbound TLS. The moment any of your services do (and ours do, to LocalStack and later AWS APIs), you need the CA bundle. `distroless/static` ships it. `scratch` does not
- **"What if I want a shell for debugging?"** Use `gcr.io/distroless/static-debian12:debug-nonroot` in dev. It adds BusyBox so `kubectl exec -it pod sh` works. Never ship `:debug-*` to prod

---

## Pitfalls we want to spot in homework

- Copy pasting the same Dockerfile into all nine services without dropping `EXPOSE` from the worker. The worker has nothing to expose
- Forgetting `CGO_ENABLED=0` and shipping a binary that segfaults on distroless with a confusing dynamic linker error
- Putting `USER nonroot` (the name) instead of `USER 65532:65532` (the UID). Kubernetes wants the numeric UID for `runAsNonRoot` checks
- Ignoring `go.sum` in `.dockerignore` and breaking reproducible builds
- Tagging images `:dev` or `:latest` and then wondering why two Pods are running different code

Bring the before and after image size for one service to next week. We will start session 3 by looking at the table.
