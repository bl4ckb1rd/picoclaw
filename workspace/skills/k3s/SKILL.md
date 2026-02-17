---
name: k3s
description: Expert instructions for managing and monitoring the bl4ckb1rd.dev k3s cluster.
---

# K3s Cluster Skill

Expert instructions for operating, monitoring, and troubleshooting the local k3s cluster.

## Cluster Architecture
- **Environment**: bl4ckb1rd.dev
- **Nodes**: Pi 5 (ARM64) and Mac nodes.
- **Services**: n8n, Grafana, Loki, Marimo, Minecraft.
- **Ingress**: Traefik with Let's Encrypt certificates.

## Common Operations

### Monitoring Health
- **Nodes**: `kubectl get nodes -o wide`
- **Pod Status**: `kubectl get pods -A`
- **Events**: `kubectl get events -A --sort-by='.lastTimestamp'`
- **Resource Usage**: `kubectl top nodes` or `kubectl top pods -A`

### Troubleshooting
- **Logs**: `kubectl logs -n <namespace> <pod-name>`
- **Describe**: `kubectl describe pod -n <namespace> <pod-name>`
- **Restart**: `kubectl rollout restart deployment <name> -n <namespace>`

### Service URLs
- **Homer Dashboard**: https://dash.bl4ckb1rd.dev
- **Grafana**: https://grafana.bl4ckb1rd.dev

## Workflow
1.  **Observe**: Use `kubectl get` to see the current state.
2.  **Analyze**: Look for pods in `Error`, `CrashLoopBackOff`, or `Pending` status.
3.  **Investigate**: Use `kubectl logs` and `kubectl describe` to find the root cause.
4.  **Remediate**: Apply fixes via `kubectl apply` or `rollout restart`.
5.  **Verify**: Confirm the service is back to `Running` status and reachable via URL.

## Security Note
- Use specific namespaces for commands whenever possible (`-n picoclaw`, `-n monitoring`, etc.).
- Never expose raw secrets or tokens in Telegram responses.
