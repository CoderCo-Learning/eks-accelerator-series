# CoderCo Technical Vocabulary (CTV)

A consolidated glossary of the technical terms introduced throughout the EKS Accelerator Series.

Each section contains the original vocabulary from its corresponding episode. The definitions have been preserved as written, with only the formatting updated for readability.

---
# Table of Contents

- [Episode 2 – Containers](#episode-2--containers)
- [Episode 3 – Networking](#episode-3--networking)
- [Episode 4 – Cluster](#episode-4--cluster)
- [Episode 5 – Karpenter](#episode-5--karpenter)
- [Episode 6 – Storage](#episode-6--storage)


---

# Episode 2 – Containers

- **Pod** — the smallest unit of work K8s schedules. One or more containers that share networking and storage. Usually one container per Pod.

- **Service (K8s) / ClusterIP** — a stable virtual IP plus a DNS name for a set of Pods. You call the Service, K8s routes to whichever Pod is ready. ClusterIP is the default type and is reachable only from inside the cluster.

- **Deployment** — a Pod template plus a replica count. K8s keeps that number of Pods running. Used for stateless apps.

- **StatefulSet** — like a Deployment, but each Pod gets a stable name (postgres-0, postgres-1) and its own disk that follows it around. We saw this in EP1.

- **Job** — runs a Pod once, tracks whether it succeeded, then exits. Used for one-shot tasks like a database migration on release.

- **CronJob** — spawns a Job on a schedule. Same idea as Linux cron, but cluster-aware.

- **Ingress** — a K8s object that says "send external traffic for this URL path to this Service". Needs an Ingress controller running in the cluster to actually do the routing.

- **HPA (Horizontal Pod Autoscaler)** — watches a metric (usually CPU) and scales the number of replicas in a Deployment up or down.

- **KEDA** — a community add-on that scales Deployments on external signals like SQS queue depth or Postgres row counts. Where the built-in HPA is "scale on CPU", KEDA is "scale on whatever metric you can name".

- **ServiceAccount** — the K8s identity a Pod runs as. IRSA annotates a ServiceAccount with an IAM role ARN so the Pod can assume that role at runtime without static keys.

- **Rolling update** — the default deploy strategy. K8s starts a new Pod, then kills an old one and repeats until done. Zero downtime if the readiness probe is honest.

- **Recreate strategy** — the opposite. Kill every old Pod first, then start the new ones. Brief downtime but you never have two versions running at the same time. Useful for singletons like the scheduler.

- **ConfigMap** — a K8s object that holds non-secret config. Mounted into Pods as env vars or files.

- **Secret** — same idea as a ConfigMap but for sensitive values. K8s does not encrypt them by default; we source them from AWS Secrets Manager via the External Secrets operator.

- **IRSA (IAM Roles for Service Accounts)** — the EKS feature that lets a Pod assume an AWS IAM role without any static access keys. The Pod gets the role via the ServiceAccount it runs as.

- **SQS (Simple Queue Service)** — AWS managed message queue. Producers send messages, consumers pull them. We use it as the event bus between order-service, payment-service, shipping-service and worker.

- **DLQ (Dead Letter Queue)** — an SQS feature. If a consumer fails to process a message a certain number of times, SQS moves it to the DLQ so it does not block the main queue.

- **ARN (Amazon Resource Name)** — the canonical AWS identifier for a resource. Looks like `arn:aws:sqs:eu-west-2:123456789012:order-events`. IRSA policies are written against ARNs.

- **NLB (Network Load Balancer)** — AWS layer-4 load balancer that forwards raw TCP traffic without inspecting it. We put one in front of our Ingress controller.

- **NAT Gateway** — AWS managed service that lets resources in private subnets initiate outbound connections to the public internet. The reason notification-service can call an external SMTP server.

- **VPC endpoint** — a private route from inside a VPC straight to an AWS service, avoiding the public internet and the NAT Gateway hop. Lets us reach SQS or S3 without a NAT.

- **Secrets Manager** — AWS managed secret store. Holds things like DB passwords and third-party API tokens. Never put these in YAML.

- **SES (Simple Email Service)** — AWS managed email sender. Would back the notification-service in real life.

- **SNS (Simple Notification Service)** — AWS managed pub/sub, also a backend for SMS. The other half of the notification-service in real life.

- **CloudWatch alarm** — AWS feature that fires (pages someone, runs a Lambda) when a metric crosses a threshold. We wire one to DLQ depth later in the series.

- **LocalStack** — a local emulator for AWS services. We use it in compose so we get a fake SQS without needing a real AWS account.

- **Long polling** — an SQS receive call that waits up to 20 seconds for a message to arrive before returning. Fewer API calls and lower latency than short polling. The worker uses this.

- **Short polling** — the opposite of long polling. The receive returns immediately even if the queue is empty. More API calls per second, higher cost.

- **Webhook** — an HTTP call made INTO our system by an external service when something happens there. Couriers POST to shipping-service when a parcel changes status.

- **JWT (JSON Web Token)** — a signed token the client carries on every request to prove who they are. We verify the signature locally instead of looking up a session in a database.

- **HMAC signature** — a cryptographic signature on a payload, computed with a shared secret. Used to prove a webhook came from who it claims to be.

- **Idempotent** — an operation you can run twice and the end state is the same as running it once. Important for SQS because the same message can be delivered more than once.

- **At-least-once delivery** — SQS guarantees a message is delivered at least once. Sometimes that means more than once. Hence the idempotency point above.

- **State machine** — an object that can only move between specific named states via specific allowed transitions. Our order goes `pending → confirmed → processing → shipped → delivered`, never directly from pending to shipped.

- **Race condition** — two operations happening at the same time produce a different (wrong) outcome than running them one after the other. The inventory reservation logic has one.

- **Distributed lock** — a mutex that works across multiple processes or machines. Usually backed by Redis or a database row lock. Cures certain race conditions.

- **Connection pool** — a set of pre-opened database connections that an app reuses. Bigger pool means more concurrent queries at the cost of more load on the database.

- **Transaction (database)** — a group of writes that either all succeed or all fail (`BEGIN ... COMMIT`). Stops the database being left half-written if the app crashes mid-flight.

- **Trace ID / distributed tracing** — a single identifier that follows one request through every service it touches, so we can stitch logs and spans into one timeline. Without it, debugging a microservices request is mostly guessing.

- **Noisy neighbour** — a workload that hogs shared resources (DB connections being the obvious one) and starves everyone else sharing them.

- **TTL (Time To Live)** — a duration after which a value expires on its own. Inventory reservations have a TTL so abandoned baskets release the stock automatically.

- **Egress** — outbound network traffic leaving a resource. For Pods specifically: making outbound calls to anything outside the cluster.

- **Traefik** — an Ingress controller. The thing that actually routes external traffic to the right Service inside the cluster.

- **cert-manager** — a K8s operator that talks to Let's Encrypt to issue and renew HTTPS certificates automatically.

- **External Secrets (operator)** — a K8s operator that syncs values from AWS Secrets Manager into K8s Secret objects so passwords never go in YAML.

- **Grafana** — open-source dashboard tool. We use it to visualise metrics from Prometheus.

- **CDN (Content Delivery Network)** — a global cache for static assets. Mentioned in this lesson only because the dashboard-api does **NOT** need one; we serve the UI from inside the Go binary.

- **Goroutine + Ticker (Go)** — a goroutine is a cheap, lightweight Go thread; a ticker is a Go construct that fires a channel on a fixed interval. The scheduler and worker both use these for background loops.
...

---

# Episode 3 – Networking 

- **AWS VPC CNI** — the default EKS pod-networking plugin. Gives every pod a real VPC IP from the node's subnet, which is why subnet sizing matters so much.

- **ENI (Elastic Network Interface)** — a virtual network card. Nodes attach several, the CNI hands their IPs to pods, and the control plane also places ENIs in your private subnets.

- **Warm pool** — the spare IPs and ENIs the CNI keeps attached and idle for fast pod startup, tuned with `WARM_ENI_TARGET` / `WARM_IP_TARGET`.

- **Prefix delegation** — a CNI mode that assigns each ENI a `/28` block of 16 IPs at once, raising pod density and slowing how fast a subnet drains.

- **Custom networking** — pointing pod IPs at a secondary VPC CIDR (often `100.64.0.0/10`) via `ENIConfig`, used when RFC1918 space is tight.

- **Service CIDR** — the cluster's virtual ClusterIP range, fixed at creation, serviced by kube-proxy, must not overlap the VPC.

- **VPC endpoint** — a private route from your VPC to an AWS service. Gateway type (S3, DynamoDB) is free, interface type (PrivateLink) costs per hour per AZ.

- **Subnet discovery tags** — `kubernetes.io/role/elb`, `kubernetes.io/role/internal-elb`, `kubernetes.io/cluster/<name>` and `karpenter.sh/discovery`. How the LB controller and Karpenter decide where to act.

- **NodeLocal DNSCache** — a per-node DNS cache that collapses CoreDNS fan-out and keeps you under the VPC resolver's per-ENI packet ceiling.

- **Pod security group** — an opt-in `SecurityGroupPolicy` attaching a real Security Group (SG) to individual pods for pod-level isolation.

---

# Episode 4 – Cluster 

- **Control plane** — the AWS-managed half of the cluster. API server, etcd, scheduler, controllers. You get an endpoint to talk to, never a server to log into.

- **Data plane** — the half you own. The EC2 nodes and the pods on them.

- **Managed node group** — an EKS-managed Auto Scaling group of nodes. EKS owns the AMI, the launch template, the join and the upgrades.

- **AL2023** — Amazon Linux 2023, the current EKS-optimised node AMI. AL2 is gone from 1.33 onward, AL2023 or Bottlerocket are the choices.

- **ENI (Elastic Network Interface)** — a virtual NIC with its own IP in a subnet. EKS uses them for control-plane reach-in and for pod IPs under the VPC CNI.

- **Cluster IAM role** — the role the EKS service assumes to manage AWS resources for you. Carries `AmazonEKSClusterPolicy`.

- **Node IAM role** — the role on the EC2 instances. Carries the worker, ECR read and CNI policies.

- **Pod Identity** — the current AWS default for per-pod IAM. A small agent addon hands a pod credentials from a mapping you create, no OIDC provider needed. Covered next session.

- **IRSA (IAM Roles for Service Accounts)** — the older per-pod IAM mechanism, through an OIDC provider and a projected token. Still required for some cases like non-EKS clusters. Covered next session alongside Pod Identity.

- **EKS Auto Mode** — AWS running the nodes, Karpenter and the core addons for you. The hands-off alternative to building the data plane yourself, which is what this series does so you understand it.

- **Addon** — a piece of cluster plumbing (kube-proxy, CoreDNS, VPC CNI, EBS CSI) installed as an EKS-managed component or a Helm chart.

- **EKS-managed addon** — an addon whose version and lifecycle AWS manages through the EKS API, tested against your Kubernetes version.

- **aws-auth ConfigMap** — the legacy IAM-to-RBAC mapping. One YAML object, easy to break, being retired.

- **Access entry** — the modern first-class EKS API for granting an IAM principal access to the cluster.

- **Access policy association** — the grant that gives an access entry a scope of permissions (cluster admin, admin, edit, view) cluster-wide or per namespace.

- **authentication_mode** — how the cluster authorises identities. `API` (access entries), `CONFIG_MAP` (legacy) or `API_AND_CONFIG_MAP` (both, for migration).

- **Endpoint access** — whether the cluster API is reachable publicly, privately or both. Lock the public side to known CIDRs.

---

# Episode 5 – Karpenter 

- **Karpenter** — an open-source node autoscaler for Kubernetes that provisions right-sized EC2 nodes directly, without Auto Scaling groups.

- **NodePool** — the Karpenter object that sets what Karpenter may buy and how it may disrupt nodes, limits included.

- **EC2NodeClass** — the Karpenter object that says how a node is built: AMI, IAM role, subnets, security groups.

- **NodeClaim** — Karpenter's record of a single node it has requested. Watch these to see provisioning happen.

- **Bin-packing** — fitting pods onto the fewest, smallest nodes that hold them. Karpenter's core saving.

- **Consolidation** — Karpenter removing or replacing nodes when the workload fits on fewer or cheaper ones.

- **Drift** — a node no longer matching its NodePool or EC2NodeClass, which triggers a replacement.

- **Expiry (`expireAfter`)** — forcing a node to be replaced after a set age so AMIs stay current.

- **Interruption** — AWS reclaiming a node, most often a Spot two-minute warning, handled through the interruption queue.

- **Interruption queue** — an SQS queue fed by EventBridge that lets Karpenter drain a node gracefully before AWS takes it.

- **Spot** — spare EC2 capacity at a steep discount that AWS can reclaim with two minutes' notice.

- **Disruption budget** — a cap on how many nodes Karpenter may disrupt at once.

- **`karpenter.sh/do-not-disrupt`** — a pod annotation that takes a pod out of scope for voluntary disruption.

- **Discovery tag** — the `karpenter.sh/discovery` tag Karpenter uses to find your subnets and security groups.

- **Pod Identity** — per-pod IAM through an agent addon and an association, the mechanism the Karpenter controller uses here.

---

# Episode 6 – Storage 

- **PersistentVolumeClaim (PVC)** — a pod's request for a disk of a given size and class.

- **PersistentVolume (PV)** — the actual disk that satisfies a claim.

- **StorageClass** — the template that provisions volumes dynamically. Names the driver and the disk settings.

- **Dynamic provisioning** — the volume being created automatically from a PVC, with no hand-made PV.

- **Access mode** — how a volume may be mounted. `ReadWriteOnce` is one node at a time, which is all EBS supports.

- **CSI driver** — the plugin that lets Kubernetes create real storage on a provider. `ebs.csi.aws.com` is the EBS one.

- **EBS volume** — an AWS disk (a block device). Lives in one Availability Zone.

- **`volumeBindingMode: WaitForFirstConsumer`** — wait until a pod is scheduled before creating the volume, so it lands in the pod's AZ.

- **`reclaimPolicy`** — what happens to the disk when the claim is deleted. `Delete` or `Retain`.

- **gp3** — the current general-purpose EBS SSD. Cheaper than gp2 and lets you tune IOPS and throughput.

- **IAM** — the AWS permission system. Every call to AWS is checked against it.

- **IAM role** — a named bundle of permissions that something can assume for a short while.

- **Assuming a role** — borrowing its permissions, getting back temporary keys that expire.

- **STS** — AWS Security Token Service. Swaps a proof of identity for temporary credentials.

- **ARN** — Amazon Resource Name, the unique id of any AWS resource.

- **IRSA (IAM Roles for Service Accounts)** — giving a pod an AWS role through the cluster OIDC provider and a trust policy.

- **OIDC provider** — the cluster registered in IAM as a signer of identity tokens, so AWS trusts tokens it issues.

- **Trust policy** — the part of an IAM role that says who is allowed to assume it. For IRSA, one service account.

- **VolumeSnapshot** — a point-in-time backup of a PVC, taken through the CSI snapshot controller.

- **VolumeSnapshotClass** — the template for taking snapshots, like a StorageClass for backups.

- **CRD (Custom Resource Definition)** — how you teach Kubernetes a new object type, like `VolumeSnapshot`.