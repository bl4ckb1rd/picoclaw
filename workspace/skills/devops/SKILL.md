---
name: devops
description: Expert instructions for infrastructure management, deployment, and system reliability.
---

# DevOps Skill

Expert instructions for infrastructure management, deployment, and system reliability.

## Expertise

- **Kubernetes (k3s)**: Managing pods, deployments, services, and ingresses.
- **CI/CD**: Configuring GitHub Actions and automated workflows.
- **Monitoring**: Working with Grafana, Loki, and Prometheus.
- **Automation**: Writing shell scripts, Makefiles, and n8n workflows.
- **Security**: Managing Let's Encrypt certificates and secret management.

## Environment Details

- **Cluster**: k3s (Pi 5 + Mac nodes).
- **Dashboard**: dash.bl4ckb1rd.dev.
- **Domain**: bl4ckb1rd.dev.

## Workflow

1.  **Status Check**: Verify the current state of the system using `kubectl` or logs.
2.  **Infrastructure as Code**: Modify YAML manifests or configuration files.
3.  **Deployment**: Apply changes and monitor rollout status.
4.  **Verification**: Confirm service health via curl or status commands.
5.  **Logging**: Check Loki/Lighstep logs if issues occur.

## Tools Preferred

- `kubectl`, `helm`, `docker`, `curl`, `pgrep`, `systemctl`.
