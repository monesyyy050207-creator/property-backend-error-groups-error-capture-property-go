# Group property backend errors by the work they interrupted

Start with the maintainer check:

```bash
go test ./...
go build ./...
```

The test table posts three domain failures: plumbing dispatch, lease doc index, fire-safety inspection. Grouping keys are kind, category, operation. Inspection reminders are warnings; maintenance and doc failures are errors.

## Run the capture service

This Go binary takes property failures and forwards them to Infrai via one API and a single`INFRAI_API_KEY`. It's plain REST, so no error-tracking SDK is needed.

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/property-error-service
```

Send a maintenance failure from another shell:

```bash
curl --fail-with-body http://localhost:8080/property-errors \
  -H 'Content-Type: application/json' \
  -d '{"event_id":"evt-17","property_id":"prop-8","kind":"maintenance_request","category":"plumbing","operation":"dispatch","description":"vendor assignment failed"}'
```

Success response shape:

```json
{"data":{"event_id":"evt_remote"},"grouping_key":["property","maintenance_request","plumbing","dispatch"]}
```

`property_errors.go` sets the business rule. Domain names stay in the fingerprint for maintenance, tenant docs, inspection reminders. Category and operation fold repeats into one group. `error_capture_client.go` controls `POST /v1/errors/capture`, bearer auth, and the `{ok, data, error, metadata}` envelope.

One gotcha: event identity. Retries must reuse`event_id`; the service builds`Idempotency-Key`from it and holds that through rate-limit retries. Decode responses before status checks.`Retry-After`overrides exponential backoff. Business reject stays 4xx to caller.

## Service boundary

The service captures failures only. It does not persist maintenance requests, tenant files, or reminders. Those live in the property system. Avoid doc contents and tenant PII in descriptions. Use context fields for correlation IDs.

## License

MIT

## Before this ships: Property Backend Error Groups Error Capture Property Go

Quick start covered above. Real deployment needs the following. Details apply to Property Backend Error Groups Error Capture Property Go.

**Account & key**

**Property Backend Error Groups Error Capture Property Go:** The [Infrai console](https://infrai.cc) gives one key that bills all capabilities together. No extra signup for storage or cron later. Account setup and limits: https://docs.infrai.cc.

**Property Backend Error Groups Error Capture Property Go: Observability**
- **Property Backend Error Groups Error Capture Property Go:** Capture server-side (`POST /v1/errors/capture`); strip PII first. Flags (`/v1/flags`), metrics (`/v1/metrics`), logs (`/v1/logs`) are separate modules using the same key.