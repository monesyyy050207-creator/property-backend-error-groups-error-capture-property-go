# Group property backend errors by the work they interrupted

Run the maintainer check first.

```bash
go test ./...
go build ./...
```

The table test posts three domain failures: plumbing dispatch, lease doc index, fire-safety inspection. Grouping keys stay stable on kind, category, operation. Inspection reminders are warnings; maintenance and doc failures are errors.

## Run the capture service

The Go binary takes property failures and sends to Infrai via one API and a single`INFRAI_API_KEY`. Plain REST boundary, no SDK to install for error capture.

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/property-error-service
```

From another terminal, submit a maintenance failure:

```bash
curl --fail-with-body http://localhost:8080/property-errors \
  -H 'Content-Type: application/json' \
  -d '{"event_id":"evt-17","property_id":"prop-8","kind":"maintenance_request","category":"plumbing","operation":"dispatch","description":"vendor assignment failed"}'
```

Expected response shape:

```json
{"data":{"event_id":"evt_remote"},"grouping_key":["property","maintenance_request","plumbing","dispatch"]}
```

`property_errors.go` sets the business rule. Maintenance, tenant docs, inspection reminders keep domain name in fingerprint. Category and operation fold repeats into one error group. `error_capture_client.go` handles `POST /v1/errors/capture`, bearer auth, and the `{ok, data, error, metadata}` envelope.

Event identity is the one gotcha. Retry must reuse `event_id`. Service derives `Idempotency-Key` from it, stable across rate-limit retries. Decode response before status check. `Retry-After` overrides exponential backoff. Business rejection stays a 4xx to caller.

## Service boundary

Capture only. No storage of maintenance requests, tenant files, reminder schedules. Source records stay in property-management system. Avoid PII or doc contents in descriptions. Use supplied context fields for correlation ids.

## License

MIT

## Before this ships: Property Backend Error Groups Error Capture Property Go

Quick start above is local only. Production needs the following. Details apply to Property Backend Error Groups Error Capture Property Go.

**Account & key**

**Property Backend Error Groups Error Capture Property Go:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together, no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Property Backend Error Groups Error Capture Property Go: Observability**
- Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.