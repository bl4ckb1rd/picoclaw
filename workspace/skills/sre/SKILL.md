---
name: sre
description: Expert instructions for Site Reliability Engineering (SRE), focus on reliability, SLIs/SLOs, and incident response.
---

# SRE Skill

Expert instructions for maintaining high availability, reliability, and performance of services.

## Core Principles

- **Error Budgets**: Balance the need for feature velocity with the requirement for reliability.
- **SLIs/SLOs**: Define and monitor Service Level Indicators and Objectives.
- **Eliminating Toil**: Automate repetitive, manual tasks to improve efficiency.
- **Incident Management**: Handle outages with structured response, communication, and resolution.
- **Post-mortems**: Conduct blameless root cause analysis to prevent recurrence.

## Expertise

- **Observability**: Using Grafana, Loki, and Prometheus to gain system insights.
- **Capacity Planning**: Predicting and managing resource requirements.
- **Change Management**: Implementing safe, automated deployment strategies (e.g., Canaries, Blue/Green).
- **Chaos Engineering**: Proactively testing system resilience.

## Workflow

1.  **Monitor**: Constantly check SLIs for breaches of SLOs.
2.  **Triage**: When an alert fires, assess the impact and urgency.
3.  **Mitigate**: Focus on restoring service as quickly as possible (roll back if necessary).
4.  **Investigate**: After restoration, identify the root cause using logs and metrics.
5.  **Prevent**: Automate the fix or improve monitoring to prevent the issue from happening again.

## Tools Preferred

- `kubectl`, `prometheus`, `grafana`, `loki`, `terraform`, `n8n`.
