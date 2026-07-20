# EP7 manifests

Postgres and Valkey, each as a headless Service, a Secret and a StatefulSet. They use the `gp3` default StorageClass from EP6.

## Apply

```bash
kubectl apply -f postgres/
kubectl apply -f valkey/
kubectl rollout status statefulset/postgres
kubectl rollout status statefulset/valkey
kubectl get pods,pvc          # postgres-0, valkey-0, data-postgres-0, data-valkey-0
```

## Prove Postgres persists

```bash
kubectl exec -it postgres-0 -- psql -U app -d app -c "create table t(x int); insert into t values (42);"
kubectl delete pod postgres-0
kubectl rollout status statefulset/postgres
kubectl exec -it postgres-0 -- psql -U app -d app -c "select * from t;"   # 42
```

## Prove Valkey works and persists

```bash
kubectl exec -it valkey-0 -- valkey-cli set hello world
kubectl delete pod valkey-0
kubectl rollout status statefulset/valkey
kubectl exec -it valkey-0 -- valkey-cli get hello      # "world", replayed from the append-only file
```

## Reach them from another pod

The DNS names are `postgres-0.postgres` and `valkey-0.valkey` in the same namespace. An app connects to those, using the password from the Secret.

## Note on the Secrets

Both Secrets are plain and hold a placeholder password. base64 is encoding, not encryption. EP8 replaces them with External Secrets, so the real password lives in AWS Secrets Manager.
