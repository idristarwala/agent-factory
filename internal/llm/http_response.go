package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"
)

// gatewayResponse transfers ownership of successful response bodies to the caller.
func gatewayResponse(client *http.Client, req *http.Request) (*http.Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, requestFailure(req.Context(), err)
	}
	if resp.StatusCode < 400 {
		return resp, nil
	}
	resp.Body.Close()
	failure := &Error{Status: resp.StatusCode, Message: http.StatusText(resp.StatusCode)}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		failure.Type, failure.Retryable = "RATE_LIMITED", true
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		failure.Type = "AUTH"
	case resp.StatusCode >= 500:
		failure.Type, failure.Retryable = "GATEWAY_5XX", true
	default:
		failure.Type = "BAD_REQUEST"
	}
	if raw := resp.Header.Get("Retry-After"); raw != "" {
		if seconds, err := strconv.ParseUint(raw, 10, 32); err == nil {
			failure.RetryAfter = time.Duration(seconds) * time.Second
		} else if when, err := http.ParseTime(raw); err == nil {
			failure.RetryAfter = max(time.Until(when), 0)
		}
	}
	return nil, failure
}

func requestFailure(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return context.Canceled
	}
	var timeout net.Error
	if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &timeout) && timeout.Timeout() {
		return &Error{Type: "TIMEOUT", Retryable: true, Message: "gateway request timed out", Cause: context.DeadlineExceeded}
	}
	return &Error{Type: "TRANSPORT", Retryable: true, Message: "gateway request failed"}
}

func decodeReply(ctx context.Context, resp *http.Response, value any) error {
	err := json.NewDecoder(resp.Body).Decode(value)
	if err == nil {
		return nil
	}
	var timeout net.Error
	if ctx.Err() != nil || errors.As(err, &timeout) && timeout.Timeout() {
		return requestFailure(ctx, err)
	}
	return &Error{Type: "BAD_RESPONSE", Status: resp.StatusCode, Message: "invalid gateway response"}
}
