// Package slocker is the client of the Superroute partner locker delivery API: authentication, every operation,
// errors with the server's code, retries for repeatable calls, idempotency keys, and webhook verification.
package slocker

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	mrand "math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Version is the SDK version.
const Version = "0.2.1"

// DefaultBaseURL is the production API host.
const DefaultBaseURL = "https://api.superroute.ca"

const maxRetryAfter = 60

var retryStatuses = map[int]bool{408: true, 425: true, 429: true, 500: true, 502: true, 503: true, 504: true}

// Operation describes one operation of the API.
type Operation struct {
	ID           string
	Method       string
	Path         string
	PathParams   []string
	Query        []string
	Body         bool
	BodyRequired bool
	Idempotent   bool
	Retry        bool
}

// Params carries an operation's path and query parameters by their spec names, and the keys "body"
// (map[string]any), "idempotencyKey" (string) and "providerId" (an integer).
type Params map[string]any

func (p Params) clone() Params {
	out := Params{}
	for k, v := range p {
		out[k] = v
	}
	return out
}

// Error is a call that failed. Act on Status and Code, never on Message. Status is 0 when no answer came back or the
// call never left; Code is the server's code (LOCKER_…), UNAUTHENTICATED, HTTP_<status>, NETWORK_ERROR or
// INVALID_ARGUMENT.
type Error struct {
	Status     int
	Code       string
	Message    string
	Errors     any
	RetryAfter *int
	Body       any
	Err        error
}

func (e *Error) Error() string { return fmt.Sprintf("%s (%d): %s", e.Code, e.Status, e.Message) }
func (e *Error) Unwrap() error { return e.Err }

// Request is what a Transport sends.
type Request struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
}

// Response is what a Transport answers.
type Response struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

// Transport sends one request; an error means no answer came back.
type Transport func(ctx context.Context, req Request) (Response, error)

// HTTPTransport sends requests with an http.Client.
func HTTPTransport(client *http.Client) Transport {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return func(ctx context.Context, r Request) (Response, error) {
		var body io.Reader
		if r.Body != nil {
			body = bytes.NewReader(r.Body)
		}
		req, err := http.NewRequestWithContext(ctx, r.Method, r.URL, body)
		if err != nil {
			return Response{}, err
		}
		for k, v := range r.Headers {
			req.Header.Set(k, v)
		}
		res, err := client.Do(req)
		if err != nil {
			return Response{}, err
		}
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return Response{}, err
		}
		headers := map[string]string{}
		for k := range res.Header {
			headers[k] = res.Header.Get(k)
		}
		return Response{Status: res.StatusCode, Headers: headers, Body: data}, nil
	}
}

// Config configures a Client.
type Config struct {
	// Token is an access token (POST /api/v1/user/login); or sign in with Login.
	Token string
	// BaseURL defaults to DefaultBaseURL.
	BaseURL string
	// ProviderID is the third-party delivery provider to act as (X-Third-Party-Provider-Id); 0 sends none.
	ProviderID int64
	// MaxRetries is how often a repeatable call (reads, deletes, creates with an idempotency key) is repeated after a
	// timeout, 408, 425, 429 or 5xx answer. nil means 2.
	MaxRetries *int
	Transport  Transport
	// Sleep waits between retries; defaults to a context-aware timer.
	Sleep     func(ctx context.Context, seconds float64) error
	UserAgent string
}

// Client is a client of the Superroute partner locker delivery API. It is safe to use from one goroutine at a time
// for sign-in; operations may run concurrently once the token is set.
type Client struct {
	Token      string
	ProviderID int64
	baseURL    string
	maxRetries int
	transport  Transport
	sleep      func(ctx context.Context, seconds float64) error
	userAgent  string
}

// New makes a Client.
func New(cfg Config) *Client {
	c := &Client{Token: cfg.Token, ProviderID: cfg.ProviderID, baseURL: strings.TrimRight(cfg.BaseURL, "/"), maxRetries: 2,
		transport: cfg.Transport, sleep: cfg.Sleep, userAgent: cfg.UserAgent}
	if c.baseURL == "" {
		c.baseURL = DefaultBaseURL
	}
	if cfg.MaxRetries != nil && *cfg.MaxRetries >= 0 {
		c.maxRetries = *cfg.MaxRetries
	}
	if c.transport == nil {
		c.transport = HTTPTransport(nil)
	}
	if c.sleep == nil {
		c.sleep = func(ctx context.Context, s float64) error {
			t := time.NewTimer(time.Duration(s * float64(time.Second)))
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		}
	}
	if c.userAgent == "" {
		c.userAgent = "slocker-go/" + Version
	}
	return c
}

// Login signs in (POST /api/v1/user/login) and keeps the access token. businessID 0 sends none.
func (c *Client) Login(ctx context.Context, email, password string, businessID int64) (any, error) {
	body := map[string]any{"email": email, "password": password}
	if businessID != 0 {
		body["business_id"] = businessID
	}
	return c.takeToken(c.send(ctx, "POST", "/v1/user/login", body, sendOptions{noAuth: true}))
}

// RefreshToken trades the token for a new one (POST /api/v1/user/refresh-token).
func (c *Client) RefreshToken(ctx context.Context) (any, error) {
	return c.takeToken(c.send(ctx, "POST", "/v1/user/refresh-token", nil, sendOptions{}))
}

// RotateToken makes a new token, the current one valid for the grace period (POST /api/v1/user/rotate-token).
// gracePeriodMinutes 0 sends none (the server's default).
func (c *Client) RotateToken(ctx context.Context, gracePeriodMinutes int) (any, error) {
	var body map[string]any
	if gracePeriodMinutes > 0 {
		body = map[string]any{"grace_period_minutes": gracePeriodMinutes}
	}
	return c.takeToken(c.send(ctx, "POST", "/v1/user/rotate-token", body, sendOptions{}))
}

// Logout revokes the token (GET /api/v1/user/logout) and forgets it.
func (c *Client) Logout(ctx context.Context) (any, error) {
	result, err := c.send(ctx, "GET", "/v1/user/logout", nil, sendOptions{})
	if err == nil {
		c.Token = ""
	}
	return result, err
}

func (c *Client) takeToken(result any, err error) (any, error) {
	if m, ok := result.(map[string]any); ok && err == nil {
		if t, ok := m["access_token"].(string); ok {
			c.Token = t
		}
	}
	return result, err
}

func invalid(msg string) error { return &Error{Status: 0, Code: "INVALID_ARGUMENT", Message: msg} }

// Call runs one operation by name. It answers the response body decoded from JSON (map[string]any, []any,
// json.Number, string, bool or nil).
func (c *Client) Call(ctx context.Context, operation string, params Params) (any, error) {
	op, ok := Operations[operation]
	if !ok {
		return nil, invalid("Unknown operation " + operation)
	}
	path := op.Path
	for _, name := range op.PathParams {
		v, ok := params[name]
		if !ok || v == nil || v == "" {
			return nil, invalid(operation + " needs " + name)
		}
		path = strings.Replace(path, "{"+name+"}", encode(text(v)), 1)
	}
	var query []string
	for _, name := range op.Query {
		if v, ok := params[name]; ok && v != nil && !isNilPointer(v) {
			query = append(query, name+"="+encode(text(v)))
		}
	}
	if len(query) > 0 {
		path += "?" + strings.Join(query, "&")
	}
	opts := sendOptions{retry: op.Retry, providerID: c.ProviderID}
	if v, ok := params["providerId"]; ok && v != nil {
		if n, err := strconv.ParseInt(text(v), 10, 64); err == nil {
			opts.providerID = n
		}
	}
	if op.Idempotent {
		key, _ := params["idempotencyKey"].(string)
		if key == "" {
			key = newUUID()
		}
		opts.idempotencyKey = key
	}
	var body any
	if op.Body {
		if b, ok := params["body"]; ok && b != nil {
			body = b
		}
	}
	return c.send(ctx, op.Method, path, body, opts)
}

type sendOptions struct {
	retry          bool
	noAuth         bool
	idempotencyKey string
	providerID     int64
}

func (c *Client) send(ctx context.Context, method, path string, body any, o sendOptions) (any, error) {
	headers := map[string]string{"Accept": "application/json", "User-Agent": c.userAgent}
	if !o.noAuth && c.Token != "" {
		headers["Authorization"] = "Bearer " + c.Token
	}
	if o.providerID != 0 {
		headers["X-Third-Party-Provider-Id"] = strconv.FormatInt(o.providerID, 10)
	}
	if o.idempotencyKey != "" {
		headers["Idempotency-Key"] = o.idempotencyKey
	}
	var data []byte
	if body != nil {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(body); err != nil {
			return nil, &Error{Code: "INVALID_ARGUMENT", Message: err.Error(), Err: err}
		}
		data = bytes.TrimRight(buf.Bytes(), "\n")
		headers["Content-Type"] = "application/json"
	}
	req := Request{Method: method, URL: c.baseURL + "/api" + path, Body: data}
	for attempt := 0; ; attempt++ {
		h := map[string]string{}
		for k, v := range headers {
			h[k] = v
		}
		req.Headers = h
		res, err := c.transport(ctx, req)
		if err != nil {
			if ctx.Err() == nil && o.retry && attempt < c.maxRetries {
				if serr := c.sleep(ctx, backoff(attempt)); serr != nil {
					return nil, &Error{Code: "NETWORK_ERROR", Message: serr.Error(), Err: serr}
				}
				continue
			}
			return nil, &Error{Code: "NETWORK_ERROR", Message: err.Error(), Err: err}
		}
		parsed := parse(res.Body)
		if res.Status >= 200 && res.Status < 300 {
			return parsed, nil
		}
		if o.retry && retryStatuses[res.Status] && attempt < c.maxRetries {
			wait := backoff(attempt)
			if ra := retryAfter(res.Headers, nil); ra != nil {
				wait = math.Min(float64(*ra), maxRetryAfter)
			}
			if serr := c.sleep(ctx, wait); serr != nil {
				return nil, &Error{Code: "NETWORK_ERROR", Message: serr.Error(), Err: serr}
			}
			continue
		}
		return nil, errorFrom(res.Status, res.Headers, parsed)
	}
}

func errorFrom(status int, headers map[string]string, body any) *Error {
	data, _ := body.(map[string]any)
	code, _ := data["code"].(string)
	if code == "" {
		if status == 401 {
			code = "UNAUTHENTICATED"
		} else {
			code = "HTTP_" + strconv.Itoa(status)
		}
	}
	msg, ok := data["message"].(string)
	if !ok {
		msg = "HTTP " + strconv.Itoa(status)
	}
	return &Error{Status: status, Code: code, Message: msg, Errors: data["errors"], RetryAfter: retryAfter(headers, body), Body: body}
}

func retryAfter(headers map[string]string, body any) *int {
	for k, v := range headers {
		if strings.EqualFold(k, "Retry-After") {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 0 {
				return &n
			}
		}
	}
	if m, ok := body.(map[string]any); ok {
		if e, ok := m["errors"].(map[string]any); ok {
			if n, ok := e["retry_after_seconds"].(json.Number); ok {
				if i, err := n.Int64(); err == nil {
					v := int(i)
					return &v
				}
			}
		}
	}
	return nil
}

func parse(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return string(raw)
	}
	return v
}

func isNilPointer(v any) bool {
	switch p := v.(type) {
	case *string:
		return p == nil
	case *int:
		return p == nil
	case *int64:
		return p == nil
	case *float64:
		return p == nil
	case *bool:
		return p == nil
	}
	return false
}

func text(v any) string {
	switch x := v.(type) {
	case bool:
		if x {
			return "1"
		}
		return "0"
	case *bool:
		return text(*x)
	case *string:
		return *x
	case *int:
		return strconv.Itoa(*x)
	case *int64:
		return strconv.FormatInt(*x, 10)
	case *float64:
		return text(*x)
	case float64:
		if x == math.Trunc(x) && math.Abs(x) < 1e15 {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case float32:
		return text(float64(x))
	case json.Number:
		return x.String()
	default:
		return fmt.Sprint(v)
	}
}

// encode percent-encodes every byte except the unreserved characters of RFC 3986.
func encode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '.' || c == '_' || c == '~' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func backoff(attempt int) float64 {
	return math.Min(8, 0.5*math.Pow(2, float64(attempt))) * (0.5 + mrand.Float64()/2)
}

func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(errors.New("slocker: no random source"))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
