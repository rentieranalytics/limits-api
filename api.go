// Package limitsapi is the contract of limits-service: NATS subjects,
// request and response bodies, and a small request-reply client.
package limitsapi

import (
	"fmt"
	"time"
)

// QueueGroup is the queue group the service subscribes with.
const QueueGroup = "limits-service"

const (
	SubjectConfigure    = "limits.v1.configure"
	SubjectSetUnlimited = "limits.v1.set_unlimited"
	SubjectSetMonthly   = "limits.v1.set_monthly"
	SubjectAddUnits     = "limits.v1.add_units"
	SubjectDelete       = "limits.v1.delete"
	SubjectConsume      = "limits.v1.consume"
	SubjectLink         = "limits.v1.link"
	SubjectUnlink       = "limits.v1.unlink"
	SubjectGet          = "limits.v1.get"
)

// ResourceValuation is the resource used for widget valuations; the widget
// identifier (CLIENT_KEY) is the subject.
const ResourceValuation = "valuation"

type Key struct {
	Resource string `json:"resource"`
	Subject  string `json:"subject"`
}

// ConfigureRequest creates a finite limit granting Monthly units from the
// current month. It fails with CodeAlreadyConfigured if the limit exists.
type ConfigureRequest struct {
	OperationID string `json:"operation_id"`
	Key
	Monthly int64 `json:"monthly"`
}

// SetUnlimitedRequest creates the limit if needed and stops enforcing it.
type SetUnlimitedRequest struct {
	OperationID string `json:"operation_id"`
	Key
}

// SetMonthlyRequest sets the allowance granted from the next month and adds
// AddCurrent units to the current month. An unlimited limit becomes finite
// immediately with Monthly+AddCurrent units available now.
type SetMonthlyRequest struct {
	OperationID string `json:"operation_id"`
	Key
	Monthly    int64 `json:"monthly"`
	AddCurrent int64 `json:"add_current"`
}

// AddUnitsRequest adds units that expire at the end of the current month.
type AddUnitsRequest struct {
	OperationID string `json:"operation_id"`
	Key
	Units int64 `json:"units"`
}

// DeleteRequest removes the limit and the link; the key becomes not_configured.
type DeleteRequest struct {
	OperationID string `json:"operation_id"`
	Key
}

// LinkRequest makes Subject use the limit of Parent (same resource) from the
// next consumption on, e.g. a widget the shared pool of its account
// ("account:<id>"). A limit of Subject itself is kept but not used while
// linked. Parent must not be linked itself.
type LinkRequest struct {
	OperationID string `json:"operation_id"`
	Key
	Parent string `json:"parent"`
}

// UnlinkRequest makes Subject use its own limit again (not_configured if it
// has none).
type UnlinkRequest struct {
	OperationID string `json:"operation_id"`
	Key
}

// GetRequest reads the limit that applies to Subject.
type GetRequest struct {
	Key
}

// ConsumeRequest takes one unit. OperationKey (the estimation ID) makes the
// call idempotent: repeating it never takes a second unit.
type ConsumeRequest struct {
	Key
	OperationKey string `json:"operation_key"`
}

const (
	StateFinite        = "finite"
	StateUnlimited     = "unlimited"
	StateNotConfigured = "not_configured"
)

const (
	DenyExhausted     = "exhausted"
	DenyNotConfigured = "not_configured"
)

type ConsumeResponse struct {
	Granted bool `json:"granted"`
	// Reason is set when Granted is false.
	Reason string `json:"reason,omitempty"`
	// Replayed is true when the unit was granted earlier for the same OperationKey.
	Replayed bool `json:"replayed,omitempty"`
}

// Limit is the state of a limit in the current month.
type Limit struct {
	Key
	// Parent is the subject Subject is linked to, if any.
	Parent string `json:"parent,omitempty"`
	// Pool is the subject whose limit applies: Parent or Subject. Set by get.
	Pool        string `json:"pool,omitempty"`
	State       string `json:"state"`
	Monthly     int64  `json:"monthly"`
	NextMonthly int64  `json:"next_monthly"`
	// Balance is present only for the finite state.
	Balance *int64 `json:"balance,omitempty"`
	Used    int64  `json:"used"`
	// Period is the current month as "YYYY-MM".
	Period  string    `json:"period"`
	ResetAt time.Time `json:"reset_at"`
	// UsageBySubject is the usage of the pool in Period by subject, e.g. by
	// widget of an account. Set by get.
	UsageBySubject map[string]int64 `json:"usage_by_subject,omitempty"`
}

// Response is the envelope of every NATS reply; exactly one field is set.
type Response[T any] struct {
	Data  *T     `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
}

const (
	CodeInvalidRequest    = "invalid_request"
	CodeInvalidAmount     = "invalid_amount"
	CodeNotConfigured     = "not_configured"
	CodeAlreadyConfigured = "already_configured"
	CodeUnlimited         = "unlimited"
	CodeOperationConflict = "operation_conflict"
	CodeInternal          = "internal"
)

// Error is a failure reported by the service. Every code except CodeInternal
// is permanent: retrying the same request returns the same error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("limits-service: %s: %s", e.Code, e.Message)
}

// Temporary reports whether retrying the request may succeed.
func (e *Error) Temporary() bool {
	return e.Code == CodeInternal
}
