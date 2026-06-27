# Research Summary: Voxel Royale Distribuido

**Domain:** distributed browser multiplayer battle royale  
**Researched:** 2026-04-24  
**Overall confidence:** MEDIUM

## Executive Summary

The project should be planned as a distributed-systems demonstration that happens to be a playable voxel battle royale. The grading criteria reward distributed architecture, implementation of required mechanisms, documentation clarity, originality, functionality and the group's ability to answer questions. Therefore, gRPC, web services, message contracts, observability and stress testing must be visible from the beginning instead of added after the game is built.

For Entrega 1, the selected mandatory requirements are RPC through gRPC and web services. The best deliverable is a running skeleton where QR room creation/join/status are exposed as HTTP services, while Gateway, Lobby and Game services communicate through gRPC contracts. Even if gameplay is initially minimal, the architecture must already prove "who exchanges messages with whom" and what each message contains.

For the full project, the most coherent architecture is a Go backend split into Gateway, Lobby and Game services, with Phaser (2D)/TypeScript on the browser client. The Game service should be authoritative, while the client handles rendering, touch controls, prediction and interpolation. Observability should include OpenTelemetry traces, Prometheus metrics and Grafana/Jaeger dashboards.

The strongest roadmap is delivery-driven: first establish contracts and the Entrega 1 proof, then build the playable loop, then scale to 50 players, then add fault handling and deployment polish for Entrega 2.

## Key Findings

**Stack:** Go + gRPC + HTTP web services + WebSocket + Phaser (2D)/TypeScript + Docker Compose + OpenTelemetry/Prometheus/Jaeger/Grafana.  
**Architecture:** Gateway handles public connections, Lobby handles room lifecycle, Game owns authoritative match state, telemetry makes distributed behavior visible.  
**Critical pitfall:** Spending too much time on visuals before the distributed architecture and Entrega 1 report are demonstrable.

## Implications for Roadmap

Suggested phase structure:

1. **Entrega 1 distributed foundation** - satisfy gRPC + web services and produce the 4-page SBC report.
2. **Playable room and client loop** - QR join, Phaser scene, controls and WebSocket snapshots.
3. **Authoritative battle royale mechanics** - tick loop, chests, weapons, damage, elimination, safe zone and ranking.
4. **Observability and stress proof** - traces, dashboards and 50-player bot simulation.
5. **Fault tolerance and deploy** - failure handling, Docker Compose hardening and VPS deployment path.
6. **Presentation and team readiness** - final report, role ownership and 10-minute demo.

**Phase ordering rationale:**

- Entrega 1 requirements are non-negotiable and should be delivered first.
- Game mechanics depend on stable room/session/network contracts.
- Stress tests and observability are only meaningful once gameplay messages exist.
- Fault tolerance and deployment build on the running distributed system.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | MEDIUM | Based on user decisions and standard ecosystem choices; exact versions should be pinned during implementation. |
| Features | HIGH | Requirements come directly from user PRD and course PDF. |
| Architecture | HIGH | Service boundaries match the chosen distributed-system demonstration. |
| Pitfalls | MEDIUM | Based on coursework risks and multiplayer project complexity. |

## Gaps to Address

- Exact deadline dates are in Canvas and were not available locally.
- Student names/roles still need to be filled in before the report.
- Exact VPS configuration on Hostinger should be validated during deploy planning.
