package httpclient

import (
	"net/http"
	"time"
)

// Transport is the interface used as a transport by *Client.
type Transport interface {
	http.RoundTripper
	// Deadline returns the Deadline for this transport.
	// The deadline represents the maximum total time for
	// the request, including DNS resolution, connection
	// establishment and the time spent reading the response
	// body.
	Deadline() time.Duration
	// SetDeadline sets the Transport deadline. Setting it
	// to 0 disables timeouts.
	SetDeadline(time.Duration)
	// UserAgent returns the default user agent sent by requests
	// without an User-Agent header set.
	UserAgent() string
	// SetUserAgent sets the default user agent.
	SetUserAgent(string)
	// Underlying returns the underlying RoundTripper. Note
	// that most of the time this will return an *http.Transport,
	// but when running in GAE the returned value will be of type
	// *urlfetch.Transport.
	Underlying() http.RoundTripper
	// SetUnderlying changes the underlying http.RoundTripper. This
	// is useful if you're using another library which provides an
	// http.RoundTripper which adds some functionality while providing
	// composition with another http.RoundTripper.
	SetUnderlying(http.RoundTripper)
}

// NewTransport returns a new Transport for the given Context.
func NewTransport(ctx Context) Transport {
	tr := &transport{}
	rt := newRoundTripper(ctx, tr)
	tr.transport = rt
	return tr
}

type transport struct {
	userAgent string
	deadline  time.Duration
	transport http.RoundTripper
}

func (t *transport) UserAgent() string {
	return t.userAgent
}

func (t *transport) SetUserAgent(ua string) {
	t.userAgent = ua
}

func (t *transport) Underlying() http.RoundTripper {
	return t.transport
}

func (t *transport) SetUnderlying(roundTripper http.RoundTripper) {
	t.transport = roundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.userAgent != "" {
		if req.Header != nil && req.Header.Get("User-Agent") == "" {
			req.Header.Add("User-Agent", t.userAgent)
		}
	}
	return t.transport.RoundTrip(req)
}
