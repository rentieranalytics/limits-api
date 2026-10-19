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
	SubjectRelease      = "limits.v1.release"
	SubjectSuspend      = "limits.v1.suspend"
	SubjectResume       = "limits.v1.resume"
	SubjectRevokeGrant  = "limits.v1.revoke_grant"
)

// ResourceValuation is the resource used for widget valuations; the widget
// identifier (CLIENT_KEY) is the subject.
const ResourceValuation = "valuation"

type Key struct {
	Resource string `json:"resource"`
	Subject  string `json:"subject"`
}

// ConfigureRequest creates a finite limit granting Monthly units from the
// current period. It fails with CodeAlreadyConfigured if the limit exists.
//
// PeriodAnchor is the day of month (1-31) periods start on, e.g. the day a
// subscription started; empty or 1 means calendar months. In months shorter
// than the anchor day a period starts on the last day of the month.
type ConfigureRequest struct {
	OperationID string `json:"operation_id"`
	Key
	Monthly      int64 `json:"monthly"`
	PeriodAnchor int   `json:"period_anchor,omitempty"`
}

// SetUnlimitedRequest creates the limit if needed and stops enforcing it.
type SetUnlimitedRequest struct {
	OperationID string `json:"operation_id"`
	Key
}

// SetMonthlyRequest sets the allowance granted from the next period and adds
// AddCurrent units to the current period. An unlimited limit becomes finite
// immediately with Monthly+AddCurrent units available now.
//
// A PeriodAnchor different from the current one starts a new period anchored
// at it at once, with Monthly+AddCurrent units and usage from zero (e.g. a
// subscription starting on another day); grants are kept. Without it the
// anchor does not change.
type SetMonthlyRequest struct {
	OperationID string `json:"operation_id"`
	Key
	Monthly      int64 `json:"monthly"`
	AddCurrent   int64 `json:"add_current"`
	PeriodAnchor *int  `json:"period_anchor,omitempty"`
}

// AddUnitsRequest adds units that expire at the end of the current month or,
// with ValidUntil, a grant of units valid until then whatever the month (e.g.
// a package valid for 12 months). Grants are used after the monthly balance,
// the one expiring first before others, and survive the monthly reset.
type AddUnitsRequest struct {
	OperationID string `json:"operation_id"`
	Key
	Units      int64      `json:"units"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`
}

// Grant is a batch of units with its own validity.
type Grant struct {
	// ID is the operation_id of the add_units call that created it.
	ID         string    `json:"id"`
	Units      int64     `json:"units"`
	Remaining  int64     `json:"remaining"`
	ValidUntil time.Time `json:"valid_until"`
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

// SuspendRequest refuses every consumption of Subject and of subjects linked
// to it (reason "suspended"), e.g. an account between subscriptions or a
// paused widget of an active account. It needs no limit of the subject's own
// and wins over a missing limit. Grants keep running out meanwhile.
// ResumeRequest lifts it; delete lifts it too.
type SuspendRequest struct {
	OperationID string `json:"operation_id"`
	Key
}

type ResumeRequest struct {
	OperationID string `json:"operation_id"`
	Key
}

// RevokeGrantRequest takes back units left in a grant, e.g. after a refund:
// all of them when Units is 0, otherwise at most Units. GrantID is the
// operation_id of the add_units call that created the grant.
type RevokeGrantRequest struct {
	OperationID string `json:"operation_id"`
	Key
	GrantID string `json:"grant_id"`
	Units   int64  `json:"units,omitempty"`
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
	DenySuspended     = "suspended"
)

// ReleaseRequest gives back the unit consumed for OperationKey, e.g. when the
// valuation produced no report. The unit goes back to the pool it was taken
// from while its period is current; afterwards it has expired. Repeating the
// call never gives back a second unit.
type ReleaseRequest struct {
	Key
	OperationKey string `json:"operation_key"`
}

const (
	ReleaseExpired     = "expired"
	ReleaseNotConsumed = "not_consumed"
)

type ReleaseResponse struct {
	// Returned is true when the unit went back to its pool.
	Returned bool `json:"returned"`
	// Reason is set when Returned is false.
	Reason string `json:"reason,omitempty"`
	// Replayed is true when the unit was released earlier for the same OperationKey.
	Replayed bool `json:"replayed,omitempty"`
}

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
	// Balance is what remains of the monthly allowance; present only for the
	// finite state.
	Balance *int64 `json:"balance,omitempty"`
	// Available is Balance plus units left in valid grants; present only for
	// the finite state.
	Available *int64 `json:"available,omitempty"`
	// Grants are the valid grants, kept also while unlimited.
	Grants []Grant `json:"grants,omitempty"`
	// Suspended means every consumption is refused: the subject or the pool it
	// is linked to is suspended.
	Suspended bool  `json:"suspended,omitempty"`
	Used      int64 `json:"used"`
	// Period is the start of the current period: "YYYY-MM" for calendar
	// months, "YYYY-MM-DD" for anchored periods.
	Period string `json:"period"`
	// PeriodAnchor is the day of month periods start on; absent for calendar
	// months.
	PeriodAnchor int       `json:"period_anchor,omitempty"`
	ResetAt      time.Time `json:"reset_at"`
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
	CodeGrantNotFound     = "grant_not_found"
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
