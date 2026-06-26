# Roadmap: Voxel Royale Distribuido

**Created:** 2026-04-24  
**Depth:** comprehensive  
**Requirements coverage:** 40/40 v1 requirements mapped

## Phases

- [ ] **Phase 1: Entrega 1 Distributed Skeleton** - Prove the course requirements gRPC + web services and produce the first SBC report.
- [ ] **Phase 2: Team Development System** - Establish by-design architecture rules, contracts, ownership and contribution workflow for 9 students.
- [ ] **Phase 3: QR Lobby and Match Start** - Let players create, join and start rooms through QR Code and web services.
- [ ] **Phase 4: Realtime Network Pipeline** - Connect browser clients to Gateway WebSocket and Game gRPC snapshots.
- [ ] **Phase 5: Playable Voxel Battle Royale** - Deliver the full 3D mobile battle royale loop.
- [ ] **Phase 6: Observability and 50-Player Stress Proof** - Make distributed behavior measurable and prove the 50-player target.
- [ ] **Phase 7: Fault Tolerance, Stateless Infra and VPS Deploy** - Harden failures and make the system transportable/deployable.
- [ ] **Phase 8: Final Report, Roles and Presentation Readiness** - Prepare Entrega 2 materials and make every student presentation-ready.

## Phase Details

### Phase 1: Entrega 1 Distributed Skeleton
**Goal**: The group has a running distributed skeleton that demonstrably satisfies gRPC/RPC and web services for the first delivery.  
**Depends on**: Nothing  
**Requirements**: COUR-01, COUR-02, COUR-03, COUR-04, ARCH-01, ARCH-02, ARCH-03, ARCH-04, ARCH-05  
**Success Criteria** (what must be TRUE):
  1. A local Docker Compose demo starts at least Gateway, Lobby and Game Go services.
  2. An HTTP web service can create or inspect a room and return structured JSON.
  3. Gateway, Lobby and Game exchange at least one meaningful request through gRPC generated from `.proto` files.
  4. Architecture documentation explains each entity, main messages and why gRPC + web services satisfy Entrega 1.
  5. A draft SBC report of up to 4 pages contains problem, architecture, chosen requirements, implementation details, challenges and student roles.
**Plans**: 5 plans

Plans:
- [x] 01-01-PLAN.md — Create Go module, Makefile, protobuf contracts and generated gRPC code.
- [x] 01-02-PLAN.md — Implement Lobby and Game gRPC services with in-memory state. (correlated logs still pending)
- [x] 01-03-PLAN.md — Implement Gateway HTTP JSON web services that call Lobby through gRPC (grpc-gateway).
- [x] 01-04-PLAN.md — Package Gateway, Lobby and Game in Docker Compose with README demo commands.
- [ ] 01-05-PLAN.md — Write architecture docs, message docs, role placeholders and Entrega 1 report draft. (partial: docs/architecture.md done; messages.md, roles.md and report draft pending)

### Phase 2: Team Development System
**Goal**: Nine students can work in parallel without breaking contracts or architecture consistency.  
**Depends on**: Phase 1  
**Requirements**: ARCH-06  
**Success Criteria** (what must be TRUE):
  1. Repository contains a by-design guide defining services, packages, contracts, testing expectations and review rules.
  2. Each service has clear ownership and boundaries that students can explain.
  3. Shared `.proto` and HTTP contracts have a documented change process.
  4. Tasks can be split across backend, frontend, observability, load testing and report groups.
**Plans**: TBD

### Phase 3: QR Lobby and Match Start
**Goal**: Players can enter a room through QR Code, appear in a lobby and start a match.  
**Depends on**: Phase 2  
**Requirements**: LOBB-01, LOBB-02, LOBB-03, LOBB-04  
**Success Criteria** (what must be TRUE):
  1. User can create a room and show a QR Code/URL to other players.
  2. Player can join from a mobile browser with only a display name.
  3. Lobby displays connected players and ready/waiting status.
  4. Lobby can trigger match start and pass players/config to the Game service.
**Plans**: TBD

### Phase 4: Realtime Network Pipeline
**Goal**: Browser clients and backend services exchange real-time gameplay inputs and snapshots reliably.  
**Depends on**: Phase 3  
**Requirements**: NETW-01, NETW-02, NETW-03, NETW-04  
**Success Criteria** (what must be TRUE):
  1. Client keeps a WebSocket connection through a full match session.
  2. Client sends sequenced movement/action inputs to Gateway.
  3. Gateway forwards inputs to Game service via gRPC.
  4. Game service publishes state snapshots that Gateway fans out to connected clients.
  5. Basic client reconciliation/interpolation makes remote players visibly move.
**Plans**: TBD

### Phase 5: Playable Voxel Battle Royale
**Goal**: The game is genuinely playable as a 3D mobile voxel battle royale with a complete match loop.  
**Depends on**: Phase 4  
**Requirements**: GAME-01, GAME-02, GAME-03, GAME-04, GAME-05, GAME-06, GAME-07, GAME-08  
**Success Criteria** (what must be TRUE):
  1. Player can move in a Babylon.js voxel arena with touch controls on mobile.
  2. Game service spawns players, chests and three weapon types.
  3. Player can open chests, collect weapons, attack and eliminate opponents.
  4. Safe zone shrinks over time and forces match conclusion within 5 minutes.
  5. Match ends with a final ranking shown to players.
**Plans**: TBD

### Phase 6: Observability and 50-Player Stress Proof
**Goal**: The system proves its distributed behavior and 50-player target with metrics, traces and load simulation.  
**Depends on**: Phase 5  
**Requirements**: NETW-05, OBSV-01, OBSV-02, OBSV-03, OBSV-04, OBSV-05  
**Success Criteria** (what must be TRUE):
  1. Traces show HTTP/WebSocket entrypoints and gRPC calls across Gateway, Lobby and Game.
  2. Metrics dashboard shows tick rate, connected players, latency, payload/bandwidth and errors.
  3. A repeatable stress command simulates 50 players joining and sending gameplay inputs.
  4. Stress-test results are captured for report and presentation.
  5. The team can explain bottlenecks and scalability tradeoffs using measured data.
**Plans**: TBD

### Phase 7: Fault Tolerance, Stateless Infra and VPS Deploy
**Goal**: The distributed system handles common failures and can run from Docker Compose locally or on the Hostinger VPS.  
**Depends on**: Phase 6  
**Requirements**: FAIL-01, FAIL-02, FAIL-03, DEPL-01, DEPL-02, DEPL-03  
**Success Criteria** (what must be TRUE):
  1. Player disconnects do not crash a match and are represented in game/lobby state.
  2. gRPC communication failures return controlled errors and observable degraded behavior.
  3. Fault-handling tests demonstrate the methods required for Entrega 2.
  4. Docker Compose starts all services, frontend and telemetry consistently.
  5. VPS deployment instructions are reproducible and include health/readiness validation.
**Plans**: TBD

### Phase 8: Final Report, Roles and Presentation Readiness
**Goal**: The group can submit and present the project clearly, with every student able to explain their contribution.  
**Depends on**: Phase 7  
**Requirements**: COUR-05, COUR-06  
**Success Criteria** (what must be TRUE):
  1. Entrega 2 report expands the first report to cover failures, chosen extra requirement, results and improvements.
  2. Source code submission includes clear instructions to run locally and, if needed, deploy.
  3. Presentation fits in 10 minutes and demonstrates architecture, gameplay, observability and stress/failure results.
  4. Each student has a documented role and prepared explanation of the part they implemented.
**Plans**: TBD

## Progress

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Entrega 1 Distributed Skeleton | 4/5 | In progress (code done, report pending) | - |
| 2. Team Development System | 0/0 | Not started | - |
| 3. QR Lobby and Match Start | 0/0 | Not started | - |
| 4. Realtime Network Pipeline | 0/0 | Not started | - |
| 5. Playable Voxel Battle Royale | 0/0 | Not started | - |
| 6. Observability and 50-Player Stress Proof | 0/0 | Not started | - |
| 7. Fault Tolerance, Stateless Infra and VPS Deploy | 0/0 | Not started | - |
| 8. Final Report, Roles and Presentation Readiness | 0/0 | Not started | - |

## Coverage Validation

All 40 v1 requirements are mapped to exactly one phase. No orphaned requirements.

---
*Roadmap created: 2026-04-24 after initialization*  
*Roadmap updated: 2026-06-26 to reflect Phase 1 implementation (plans 01-01..01-04 done, 01-05 partial)*
