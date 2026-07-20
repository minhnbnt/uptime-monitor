# Uptime Monitor — Kubernetes (Helm)

This chart deploys the 6 uptime-monitor microservices into Kubernetes, with
Traefik as the ingress controller. Stateful infrastructure (Postgres, PgBouncer,
Redis, Temporal, Mailpit) runs **outside** the cluster in a separate docker
compose, reached from the cluster via `ExternalName` services.

## Layout

```
helm/uptime-monitor/        # the Helm chart
  values.yaml               # structured per-service config + global options
  templates/                # namespace, externalname, traefik, applications
compose.infra.yml           # stateful infra to run on the Docker host
```

## 1. Start infra (outside the cluster)

```bash
docker compose -f compose.infra.yml up -d
```

The chart's `global.externalHost` (default `host.docker.internal`) points the
in-cluster `ExternalName` services (`postgres`, `redis`, `temporal`,
`mailpit`, `pgbouncer`) at this compose stack.

## 2. Install the chart

```bash
helm install uptime-monitor helm/uptime-monitor \
  --namespace uptime-monitor --create-namespace
```

Override any value, e.g. point at a different infra host or image tag:

```bash
helm install uptime-monitor helm/uptime-monitor \
  --set global.externalHost=10.0.0.5 \
  --set global.image.tag=v1.2.3
```

## 3. Access

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
