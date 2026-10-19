# limits-api

Contract of limits-service: NATS subjects, request and response bodies, error
codes and a small Go client for NATS request-reply. It holds no service logic.

```go
import limitsapi "github.com/rentieranalytics/limits-api"

c := limitsapi.NewClient(nc) // *nats.Conn
res, err := c.Consume(ctx, limitsapi.ConsumeRequest{
	Key:          limitsapi.Key{Resource: limitsapi.ResourceValuation, Subject: widgetID},
	OperationKey: estimationID,
})
```

## Subjects

| Subject | Request | Reply `data` |
|---|---|---|
| `limits.v1.configure` | `operation_id, resource, subject, monthly[, period_anchor]` | `Limit` |
| `limits.v1.set_unlimited` | `operation_id, resource, subject` | `Limit` |
| `limits.v1.set_monthly` | `operation_id, resource, subject, monthly, add_current[, period_anchor]` | `Limit` |
| `limits.v1.add_units` | `operation_id, resource, subject, units[, valid_until]` | `Limit` |
| `limits.v1.delete` | `operation_id, resource, subject` | `Limit` |
| `limits.v1.link` | `operation_id, resource, subject, parent` | `Limit` |
| `limits.v1.unlink` | `operation_id, resource, subject` | `Limit` |
| `limits.v1.get` | `resource, subject` | `Limit` |
| `limits.v1.consume` | `resource, subject, operation_key` | `ConsumeResponse` |
| `limits.v1.release` | `resource, subject, operation_key` | `ReleaseResponse` |

Every reply is `{"data": ...}` or `{"error": {"code", "message"}}`. Only the
`internal` code is temporary. A client-side timeout or "no responders" means the
outcome is unknown: retry with the same `operation_id` / `operation_key`, which
makes every call idempotent.

The JSON shape is pinned by tests in this module; changing it is a breaking
change for every caller.
