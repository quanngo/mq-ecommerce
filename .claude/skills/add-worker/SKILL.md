---
name: add-worker
description: Add, enable, or reschedule a cron worker in the Go backend — implement the job on a service, register it in server/workers.go mappingWorkers(), add its config/workers.yml entry, and wire any new dependency in server/base.go. Covers the valid schedule syntax and the failure modes that silently disable or crash workers. Use when asked to add a cronjob/worker/scheduled job, change how often a job runs, or turn a job on/off.
---

Most of the backend's work is done by cron workers (`robfig/cron/v3`), not by
HTTP endpoints. They start with the server (`go run main.go`). `run-job` is a
separate one-shot command that does **not** read `workers.yml`.

## How a worker is wired (read these before changing anything)

- `config/workers.yml` holds the entries: `name`, `time-run`, `function`,
  `enable`. Every `*.yml` in `config/` is merged into `utils.AppConfig`, and
  this list becomes `AppConfig.Workers`.
- `server/workers.go`: `RunCronJob()` loops over enabled entries and calls
  `c.AddFunc(time-run, mappingWorkers()[function])`. `mappingWorkers()` maps
  each `function` string to a `func()` method.
- `server/base.go`: `InitializeWorkers()` builds the services that
  `workerInstance` holds. Wiring is duplicated across `Initialize()`,
  `InitializeWorkers()` and `InitializeJob()`, so add a new dependency to each
  one that needs it.

## Steps

1. **Implement the job** as an exported no-argument method on the service
   that owns the logic (fan-out jobs across shops live in
   `component/services/platform_service.go`), and add it to that service's
   interface. Copy the existing `Cronjob*` methods:
   - Start with `defer func() { if r := recover(); r != nil { utils.ShowErrorLogs(...) } }()`.
     The cron has no `Recover` wrapper, so a panic in a job **kills the whole
     process**, API server included.
   - If the job loops over shops, use the same bounded concurrency
     (`sync.WaitGroup` + `sem := make(chan struct{}, 10)`) with a `recover`
     inside each goroutine.
   - Log start and finish with `utils.ShowInfoLogs("[<Service>] [<Method>] ...")`.
   - Never log a whole `models.MarketToken`. It has no json tags, so
     `MarshalToJson(shop)` prints `AccessToken` and `RefreshToken`. Log
     `shop.MarketPlace` and `shop.ShopID` instead.

2. **Register it** in `mappingWorkers()` in `server/workers.go`:
   `mapHandler["myNewJob"] = w.platformSvc.CronjobMyNewJob`. If it lives on a
   service that `workerInstance` doesn't hold yet, add the field and set it in
   `InitializeWorkers()`.
   - The `function` string in YAML must match this key exactly. If it
     doesn't, `AddFunc` registers a **nil func** that panics, and kills the
     process, the first time it fires.

3. **Add the YAML entry** to `config/workers.yml` (it is tracked; no secrets):
   ```yaml
   - name: my_new_job
     time-run: "@every 5m"
     function: myNewJob
     enable: true
   ```
   `time-run` uses the **standard 5-field** parser (`cron.New()` without
   `WithSeconds`):
   - Valid: `"@every 30s"`, `"@every 1h"`, `"@hourly"`, `"*/5 * * * *"`
     (minute hour dom month dow).
   - **Invalid**: `"@every 1H"` (units are lowercase: `s m h`) and 6-field
     specs like `"0 1 * * * *"`. `RunCronJob` ignores the `AddFunc` error and
     still logs `Init worker ... successful`, so an invalid spec means the job
     **never runs and nothing reports it**. The commented-out
     `checkExpiredToken*` entries in `workers.yml` use `@every 1H` and have
     this bug.

4. **Choose an interval longer than the job's worst-case runtime.** The cron
   starts a new run even if the previous one hasn't finished, and the HTTP
   clients have no timeouts, so a slow marketplace API can pile runs up. If
   the job can't be made idempotent and fast, say so to the developer rather
   than picking `@every 30s` by default.

5. **Verify**:
   - Run the `backend-check` skill.
   - Start the server and confirm the log line
     `Init worker [my_new_job], time [...] successful`, then confirm the
     job's own start/finish lines appear at the expected interval. "Init
     successful" alone does not prove the spec is valid (see step 3).
   - To turn a job off, set `enable: false`. Don't delete the entry or its
     `mappingWorkers()` key.
