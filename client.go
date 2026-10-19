package limitsapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// DefaultTimeout bounds a request when ctx has no deadline.
const DefaultTimeout = 5 * time.Second

// Client calls limits-service over NATS request-reply.
//
// Errors returned as *Error come from the service. Any other error (timeout,
// no responders, broken connection) means the outcome is unknown; retrying
// with the same operation ID or key is safe.
type Client struct {
	nc *nats.Conn
}

func NewClient(nc *nats.Conn) *Client {
	return &Client{nc: nc}
}

func (c *Client) Consume(ctx context.Context, req ConsumeRequest) (ConsumeResponse, error) {
	return call[ConsumeResponse](ctx, c.nc, SubjectConsume, req)
}

func (c *Client) Release(ctx context.Context, req ReleaseRequest) (ReleaseResponse, error) {
	return call[ReleaseResponse](ctx, c.nc, SubjectRelease, req)
}

func (c *Client) Suspend(ctx context.Context, req SuspendRequest) (Limit, error) {
	return call[Limit](ctx, c.nc, SubjectSuspend, req)
}

func (c *Client) Resume(ctx context.Context, req ResumeRequest) (Limit, error) {
	return call[Limit](ctx, c.nc, SubjectResume, req)
}

func (c *Client) Configure(ctx context.Context, req ConfigureRequest) (Limit, error) {
	return call[Limit](ctx, c.nc, SubjectConfigure, req)
}

func (c *Client) SetUnlimited(ctx context.Context, req SetUnlimitedRequest) (Limit, error) {
	return call[Limit](ctx, c.nc, SubjectSetUnlimited, req)
}

func (c *Client) SetMonthly(ctx context.Context, req SetMonthlyRequest) (Limit, error) {
	return call[Limit](ctx, c.nc, SubjectSetMonthly, req)
}

func (c *Client) AddUnits(ctx context.Context, req AddUnitsRequest) (Limit, error) {
	return call[Limit](ctx, c.nc, SubjectAddUnits, req)
}

func (c *Client) Delete(ctx context.Context, req DeleteRequest) (Limit, error) {
	return call[Limit](ctx, c.nc, SubjectDelete, req)
}

func (c *Client) Link(ctx context.Context, req LinkRequest) (Limit, error) {
	return call[Limit](ctx, c.nc, SubjectLink, req)
}

func (c *Client) Unlink(ctx context.Context, req UnlinkRequest) (Limit, error) {
	return call[Limit](ctx, c.nc, SubjectUnlink, req)
}

func (c *Client) Get(ctx context.Context, req GetRequest) (Limit, error) {
	return call[Limit](ctx, c.nc, SubjectGet, req)
}

func call[T any](ctx context.Context, nc *nats.Conn, subject string, req any) (T, error) {
	var zero T
	body, err := json.Marshal(req)
	if err != nil {
		return zero, fmt.Errorf("encode %s request: %w", subject, err)
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}
	msg, err := nc.RequestWithContext(ctx, subject, body)
	if err != nil {
		return zero, fmt.Errorf("request %s: %w", subject, err)
	}
	var resp Response[T]
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return zero, fmt.Errorf("decode %s response: %w", subject, err)
	}
	if resp.Error != nil {
		return zero, resp.Error
	}
	if resp.Data == nil {
		return zero, errors.New("limits-service: empty response")
	}
	return *resp.Data, nil
}
