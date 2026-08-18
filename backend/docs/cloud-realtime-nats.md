# Cloud Center realtime: NATS production runbook

Cloud Center scheme cards and aggregate statistics use NATS Core as an ephemeral snapshot transport. PostgreSQL remains authoritative. NATS failure must degrade browser freshness only; it must not stop scheme calculation, betting, settlement, or database writes.

## 1. Production topology

Recommended production layout:

```text
                         client TCP 4222
  API-1 / API-2  -------------------------------+
  Worker-1 / Worker-2 --------------------------+---- NATS-1 10.0.0.11
                                                 |       | route TCP 6222
                                                 +---- NATS-2 10.0.0.12
                                                 |       | route TCP 6222
                                                 +---- NATS-3 10.0.0.13

  browsers --HTTPS/WSS--> load balancer --> API-1 / API-2
  API and Worker nodes ---------------------> one PostgreSQL primary
```

- Open TCP `4222` from API/Worker nodes to all NATS nodes.
- Open TCP `6222` only between NATS nodes.
- Do not expose `4222`, `6222`, or monitoring port `8222` to the public Internet.
- NATS Core is intentionally used without JetStream. The PostgreSQL cursor reconciler republishes missed current state.

Example `/etc/nats/nats-server.conf` for NATS-1; change `server_name`, `listen`, and routes on the other two nodes:

```conf
server_name: nats-1
listen: 0.0.0.0:4222

authorization {
  users = [
    {
      user: "caipiao_api"
      password: "REPLACE_FROM_SECRET_STORE"
      permissions: {
        publish: ["caipiao.client.*.scheme", "caipiao.client.*.cloud_stats"]
        subscribe: ["caipiao.client.*.scheme", "caipiao.client.*.cloud_stats"]
      }
    },
    {
      user: "caipiao_worker"
      password: "REPLACE_FROM_SECRET_STORE"
      permissions: {
        publish: ["caipiao.client.*.scheme", "caipiao.client.*.cloud_stats"]
        subscribe: []
      }
    }
  ]
}

cluster {
  name: caipiao-prod
  listen: 0.0.0.0:6222
  authorization {
    user: "route_user"
    password: "REPLACE_ROUTE_SECRET"
  }
  routes = [
    nats-route://route_user:REPLACE_ROUTE_SECRET@10.0.0.12:6222
    nats-route://route_user:REPLACE_ROUTE_SECRET@10.0.0.13:6222
  ]
}

http: 127.0.0.1:8222
```

Use a secret manager or NATS operator/account credentials in production. Do not commit passwords, tokens, or `.creds` files. When using NATS credentials files, create separate API and Worker credentials with the permissions above and set file mode `0600` owned by the backend service account.

## 2. Backend role configuration

Every node runs the same backend binary. There is no implicit `SERVER_ROLE`; the existing worker switches define the role.

Common realtime environment for all nodes:

```env
CLOUD_REALTIME_ENABLED=true
CLOUD_REALTIME_BUS=nats
NATS_URL=nats://10.0.0.11:4222,nats://10.0.0.12:4222,nats://10.0.0.13:4222
NATS_SUBJECT_PREFIX=caipiao
CLOUD_REALTIME_COALESCE_MS=200
CLOUD_STATS_COALESCE_MS=1000
CLOUD_RECONCILE_INTERVAL_MS=5000
CLOUD_RECONCILE_BATCH=500
```

API node role:

```env
WS_ENABLED=true
SCHEME_WORKER_ENABLED=false
NATS_CREDENTIALS_FILE=/etc/caipiao/nats/api.creds
```

Worker node role:

```env
WS_ENABLED=false
SCHEME_WORKER_ENABLED=true
NATS_CREDENTIALS_FILE=/etc/caipiao/nats/worker.creds
```

If user/password authentication is used instead, set `NATS_USER` and `NATS_PASSWORD` and leave `NATS_CREDENTIALS_FILE` and `NATS_TOKEN` blank. Authentication precedence is credentials file, token, then user/password. The Worker credential shown above is publish-only; API nodes require both publish and subscribe because HTTP mutations can originate on API nodes and connected browsers consume snapshots there.

Other background workers (`GUAJI_ENABLED`, draw/history synchronization, and related services) retain their existing deployment switches. Confirm their ownership before adding Worker replicas; this realtime feature does not redefine those roles.

Frontend build environment:

```env
VITE_WS_ENABLED=true
VITE_CLOUD_REALTIME_ENABLED=true
```

## 3. Rollout

1. Deploy and verify the three-node NATS cluster. Confirm every API/Worker host can connect to every `:4222` endpoint.
2. Install dedicated API and Worker credentials. Verify their publish/subscribe permissions with a non-production subject prefix first.
3. Deploy the backend with `CLOUD_REALTIME_ENABLED=true` before deploying the new client. Legacy `client.scheme.instance.updated` invalidation remains available during this phase.
4. Verify backend health and authenticated administrator diagnostics:

   ```bash
   curl -fsS http://127.0.0.1:8080/api/v1/health
   curl -fsS -H "Authorization: Bearer $ADMIN_TOKEN" \
     http://127.0.0.1:8080/api/v1/admin/diagnostics/cloud-realtime
   ```

   Expected diagnostics: `enabled=true`, `bus.connected=true`; publisher queue sizes remain bounded; `publishErrors` and scanner errors do not continually increase; Hub subscribed-member count follows connected users. Do not paste diagnostic output into tickets until it has been checked for account or infrastructure data.
5. Deploy the client built with both realtime flags enabled.
6. In browser DevTools, open Network, filter Fetch/XHR by `/client/cloud/schemes/running` and `/client/cloud/schemes/stats`, then leave Cloud Center online with a stable WebSocket for at least five minutes. Initial load and one subscribed-cycle reconciliation are expected. After that, there must be zero periodic background requests to either endpoint.
7. Force one browser WebSocket close. After subscription recovery, verify exactly one pair of reconciliation requests, then again zero periodic requests.
8. Observe diagnostics and application errors through one complete scheme start/update/settlement cycle before widening rollout.

## 4. Read-only and synthetic smoke checks

The smoke command subscribes only to the selected numeric member subjects by default. It prints event kind, schema version, and counts; it never prints payloads or credentials. It succeeds only after observing both a scheme snapshot and a statistics snapshot before the timeout.

Read-only observation during an active scheme mutation:

```bash
cd /opt/caipiao/backend
NATS_CREDENTIALS_FILE=/etc/caipiao/nats/api.creds \
go run ./cmd/cloud-realtime-smoke \
  -nats nats://10.0.0.11:4222,nats://10.0.0.12:4222,nats://10.0.0.13:4222 \
  -prefix caipiao -member-id 7 -timeout 15s
```

Synthetic publication is opt-in and writes two empty diagnostic snapshots to the selected member subjects. Use only with a dedicated test member and explicit `-publish`:

```bash
NATS_CREDENTIALS_FILE=/etc/caipiao/nats/api.creds \
go run ./cmd/cloud-realtime-smoke -nats nats://10.0.0.11:4222 \
  -prefix caipiao-smoke -member-id 7 -timeout 15s -publish
```

The command exits non-zero when NATS does not connect or both event types are not observed before the timeout.

## 5. Failure diagnosis

| Symptom | Check | Meaning/action |
|---|---|---|
| `/health` says realtime is degraded | `bus.connected`, `lastDisconnectedAt`, NATS service/routes | Browser realtime is unavailable; application HTTP and workers should remain healthy. |
| Browser closes with code `1012`, reason `realtime_bus_unavailable` | NATS connectivity and credentials | Expected protective close; browser reconnects and performs one REST reconciliation after subscription readiness. |
| Browser closes with code `4010` | Hub backpressure counter | Only the slow connection is closed; investigate client/network slowness. |
| Publisher queues or errors grow | DB projection latency, NATS publish errors | PostgreSQL remains authoritative; fix dependency and let the scanner republish current state. |
| Cards appear stale after reconnect | scanner leader/cursor/error fields; one-shot REST requests | Confirm REST reconciliation and cursor scanner are both progressing. |
| Cross-member delivery is suspected | numeric member subject and authenticated member ID | Stop rollout and run the guarded cluster isolation test; account names must never be used as subjects. |

## 6. Rollback

Rollback does not require stopping betting workers or removing NATS:

1. Set `CLOUD_REALTIME_ENABLED=false` on every backend node and restart the backend. This restores the in-process legacy `client.scheme.instance.updated` invalidation path.
2. Rebuild and deploy the client with `VITE_CLOUD_REALTIME_ENABLED=false`. The Cloud Center returns to legacy REST polling (15 seconds while disconnected, 60 seconds while connected).
3. Keep `VITE_WS_ENABLED=true` unless the entire WebSocket service must also be rolled back.
4. Verify `/running` and `/stats` polling is present and Cloud Center actions still update cards.

Do not remove the legacy invalidation path until all deployed client versions understand `client.scheme.instances.snapshot`, all backend nodes have remained realtime healthy for the agreed observation window, forced reconnect tests show one reconciliation cycle, and monitoring confirms no legacy-only clients remain. Removal must be a separately reviewed release.

## 7. Acceptance evidence

The following checks require a live two-API/two-Worker environment and are not proven by unit tests:

1. Stable WebSocket for five minutes: zero periodic `/schemes/running` and `/schemes/stats` requests after initial reconciliation.
2. One forced socket close: exactly one reconciliation cycle after subscription recovery.
3. One hundred updates to one instance inside 200 ms: one scheme snapshot publication.
4. NATS restart while schemes run: betting and settlement continue; browser sockets close and reconcile after recovery.
5. Slow WebSocket client: only that connection closes with code `4010`.
6. Two members on different API nodes: no cross-member event.
7. Mutation-commit to browser-handler snapshot delivery P95 is below one second.

Record timestamps, node names, browser HAR/counters, diagnostics snapshots, and result for every item. Never mark an item passed when the required topology or measurement tooling was unavailable.
