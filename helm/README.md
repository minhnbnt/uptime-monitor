# Uptime Monitor — Kubernetes (Helm)

This chart deploys the 6 uptime-monitor microservices into Kubernetes, with
Traefik as the ingress controller. Stateful infrastructure (Postgres, PgBouncer,
Redis, Temporal, Mailpit) runs **outside** the cluster in a separate docker
compose, reached from the cluster via `ExternalName` services.

## Layout

```
helm/uptime-monitor/        # the Helm chart
  values.yaml               # structured per-service config (gitignored)
  values.example.yaml       # example config, copy to values.yaml
  secrets.yaml              # sensitive values: DB passwords, JWT keys (gitignored)
  secrets.example.yaml      # example secrets, copy to secrets.yaml
  templates/                # namespace, externalname, traefik, applications, secrets
compose.infra.yml           # stateful infra to run on the Docker host
```

## 1. Start infra (outside the cluster)

```bash
docker compose -f compose.infra.yml up -d
```

The chart's `global.externalHost` (default `host.docker.internal`) points the
in-cluster `ExternalName` services (`postgres`, `redis`, `temporal`,
`mailpit`, `pgbouncer`) at this compose stack.

## 2. Prepare values & secrets

```bash
cp helm/uptime-monitor/values.example.yaml helm/uptime-monitor/values.yaml
cp helm/uptime-monitor/secrets.example.yaml helm/uptime-monitor/secrets.yaml
# Edit values.yaml (set externalHost, image tag, replicas, etc.)
# Edit secrets.yaml (set real DB passwords, JWT keys, Redis passwords)
```

## 3. Install the chart

```bash
helm install uptime-monitor helm/uptime-monitor \
  --namespace uptime-monitor --create-namespace \
  -f helm/uptime-monitor/values.yaml \
  -f helm/uptime-monitor/secrets.yaml
```

Override any value, e.g. point at a different infra host or image tag:

```bash
helm upgrade uptime-monitor helm/uptime-monitor \
  -n uptime-monitor \
  -f helm/uptime-monitor/values.yaml \
  -f helm/uptime-monitor/secrets.yaml \
  --set global.externalHost=10.0.0.5 \
  --set global.image.tag=v1.2.3
```

## 4. Access

Traefik is exposed as a `NodePort` service (30080 = HTTP, 30081 = admin UI):

```bash
# from the cluster node
curl http://localhost:30080/api/v1/auth/...
# Traefik dashboard
open http://localhost:30081
```

## Notes

- gRPC servers (`server-service:50051`, `ontime-service:50052`) are exposed via
  **headless** Services (`clusterIP: None`) so gRPC clients get stable per-pod
  DNS without cluster-IP load balancing.
- All app deployments are stateless (no PVC).
- Postgres is a single instance with multiple databases (`auth`, `server`,
  `analytics`, `notification`).
