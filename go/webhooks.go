package slocker

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DefaultToleranceSeconds is how far a webhook's timestamp may be from now.
const DefaultToleranceSeconds = 300

// WebhookVerificationError is a webhook that must be refused. Reason is one of MISSING_HEADER, INVALID_TIMESTAMP,
// TIMESTAMP_OUT_OF_TOLERANCE, DIGEST_MISMATCH, SIGNATURE_MISMATCH or INVALID_PAYLOAD.
type WebhookVerificationError struct{ Reason string }

func (e *WebhookVerificationError) Error() string { return "slocker: webhook refused: " + e.Reason }

// WebhookEvent is a verified webhook: partner_locker.* or carrier_package.distribution.*
type WebhookEvent struct {
	EventID    string
	EventType  string
	Livemode   bool
	Test       bool
	OccurredAt string
	Data       map[string]any
	Payload    map[string]any
	DeliveryID string
	KeyID      string
	Headers    map[string]string
}

// VerifyOptions tunes Verify. Zero values mean the defaults.
type VerifyOptions struct {
	ToleranceSeconds int
	// Now is the current Unix time; 0 means time.Now().
	Now int64
}

var digits = regexp.MustCompile(`^[0-9]{1,12}$`)

func refuse(reason string) error { return &WebhookVerificationError{Reason: reason} }

func mac(secret string, msg []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(msg)
	return base64.StdEncoding.EncodeToString(m.Sum(nil))
}

func equal(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }

func checkTimestamp(v string, tolerance int, now int64) error {
	if !digits.MatchString(v) {
		return refuse("INVALID_TIMESTAMP")
	}
	ts, _ := strconv.ParseInt(v, 10, 64)
	if now == 0 {
		now = time.Now().Unix()
	}
	d := now - ts
	if d < 0 {
		d = -d
	}
	if d > int64(tolerance) {
		return refuse("TIMESTAMP_OUT_OF_TOLERANCE")
	}
	return nil
}

// Verify checks a webhook request and answers its event. body must be the raw request body exactly as received;
// secrets the endpoint's signing secret, or several during a rotation window. For an http.Request, pass
// HeaderMap(r.Header).
func Verify(body []byte, headers map[string]string, secrets []string, opts *VerifyOptions) (*WebhookEvent, error) {
	tolerance, now := DefaultToleranceSeconds, int64(0)
	if opts != nil {
		if opts.ToleranceSeconds > 0 {
			tolerance = opts.ToleranceSeconds
		}
		now = opts.Now
	}
	h := map[string]string{}
	for k, v := range headers {
		h[strings.ToLower(k)] = v
	}
	native := false
	switch {
	case h["x-webhook-signature"] != "":
		for _, n := range []string{"x-webhook-timestamp", "x-webhook-delivery-id", "x-webhook-content-digest", "x-webhook-signature"} {
			if h[n] == "" {
				return nil, refuse("MISSING_HEADER")
			}
		}
		ts := h["x-webhook-timestamp"]
		if err := checkTimestamp(ts, tolerance, now); err != nil {
			return nil, err
		}
		sum := sha256.Sum256(body)
		if !equal(h["x-webhook-content-digest"], "sha256="+base64.StdEncoding.EncodeToString(sum[:])) {
			return nil, refuse("DIGEST_MISMATCH")
		}
		msg := append([]byte(ts+"\n"+h["x-webhook-delivery-id"]+"\n"), body...)
		matched := false
		for _, s := range secrets {
			if s != "" && equal(h["x-webhook-signature"], "v1="+mac(s, msg)) {
				matched = true
			}
		}
		if !matched {
			return nil, refuse("SIGNATURE_MISMATCH")
		}
		native = true
	case h["webhook-signature"] != "":
		if h["webhook-id"] == "" || h["webhook-timestamp"] == "" {
			return nil, refuse("MISSING_HEADER")
		}
		ts := h["webhook-timestamp"]
		if err := checkTimestamp(ts, tolerance, now); err != nil {
			return nil, err
		}
		msg := append([]byte(h["webhook-id"]+"."+ts+"."), body...)
		matched := false
		for _, s := range secrets {
			if s == "" {
				continue
			}
			expected := "v1," + mac(s, msg)
			for _, r := range strings.Split(h["webhook-signature"], " ") {
				if r != "" && equal(r, expected) {
					matched = true
				}
			}
		}
		if !matched {
			return nil, refuse("SIGNATURE_MISMATCH")
		}
	default:
		return nil, refuse("MISSING_HEADER")
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var payload map[string]any
	if err := dec.Decode(&payload); err != nil || payload == nil {
		return nil, refuse("INVALID_PAYLOAD")
	}
	if _, ok := payload["specversion"]; ok {
		if data, ok := payload["data"].(map[string]any); ok {
			payload = data
		}
	}
	eventType, ok := payload["event_type"].(string)
	if !ok {
		return nil, refuse("INVALID_PAYLOAD")
	}
	testHeader := h["x-webhook-test"] == "1"
	livemode, ok := payload["livemode"].(bool)
	if !ok {
		livemode = !testHeader
	}
	ev := &WebhookEvent{EventType: eventType, Livemode: livemode, Test: testHeader || !livemode, Payload: payload, Headers: h, Data: map[string]any{}}
	if id, ok := payload["event_id"].(string); ok {
		ev.EventID = id
	} else if h["x-webhook-id"] != "" {
		ev.EventID = h["x-webhook-id"]
	} else {
		ev.EventID = h["webhook-id"]
	}
	if at, ok := payload["occurred_at"].(string); ok {
		ev.OccurredAt = at
	}
	if d, ok := payload["data"].(map[string]any); ok {
		ev.Data = d
	}
	if native {
		ev.DeliveryID = h["x-webhook-delivery-id"]
		ev.KeyID = h["x-webhook-key-id"]
	}
	return ev, nil
}

// HeaderMap answers the first value of every header.
func HeaderMap(h http.Header) map[string]string {
	out := map[string]string{}
	for k := range h {
		out[k] = h.Get(k)
	}
	return out
}

// SignOptions are the optional parts of Sign.
type SignOptions struct {
	Timestamp  int64
	DeliveryID string
	KeyID      string
	EventID    string
	Test       bool
}

// Sign answers the headers Superroute would send with body: for testing your own webhook receiver.
func Sign(secret string, body []byte, o SignOptions) map[string]string {
	ts := o.Timestamp
	if ts == 0 {
		ts = time.Now().Unix()
	}
	did := o.DeliveryID
	if did == "" {
		did = newUUID()
	}
	t := strconv.FormatInt(ts, 10)
	sum := sha256.Sum256(body)
	h := map[string]string{
		"Content-Type":             "application/json",
		"X-Webhook-Delivery-Id":    did,
		"X-Webhook-Timestamp":      t,
		"X-Webhook-Content-Digest": "sha256=" + base64.StdEncoding.EncodeToString(sum[:]),
		"X-Webhook-Signature":      "v1=" + mac(secret, append([]byte(t+"\n"+did+"\n"), body...)),
	}
	if o.EventID != "" {
		h["X-Webhook-Id"] = o.EventID
	}
	if o.KeyID != "" {
		h["X-Webhook-Key-Id"] = o.KeyID
	}
	if o.Test {
		h["X-Webhook-Test"] = "1"
	}
	return h
}
