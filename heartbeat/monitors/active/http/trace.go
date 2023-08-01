package http

import (
	"context"
	"crypto/tls"
	"net"
	"net/http/httptrace"
	"time"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// traceInfo struct
//_______________________________________________________________________

//traceInfo is the connection timecase about timestamp and timeduration.

type traceInfos struct {
	// traceDuration
	clientTrace
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// traceDuration struct
//_______________________________________________________________________

// traceDuration struct is used provide request trace info such as DNS lookup
// duration, Connection obtain duration, Server processing duration, etc.
//
// Since v2.0.0
// type traceDuration struct {
// 	// DNSLookup is a duration that transport took to perform
// 	// DNS lookup.
// 	DNSLookup time.Duration

// 	// ConnTime is a duration that took to obtain a successful connection.
// 	ConnTime time.Duration

// 	// TCPConnTime is a duration that took to obtain the TCP connection.
// 	TCPConnTime time.Duration

// 	// TLSHandshake is a duration that TLS handshake took place.
// 	TLSHandshake time.Duration

// 	// ServerTime is a duration that server took to respond first byte.
// 	ServerTime time.Duration

// 	// ResponseTime is a duration since first response byte from server to
// 	// request completion.
// 	ResponseTime time.Duration

// 	// TotalTime is a duration that total request took end-to-end.
// 	TotalTime time.Duration

// 	// IsConnReused is whether this connection has been previously
// 	// used for another HTTP request.
// 	IsConnReused bool

// 	// IsConnWasIdle is whether this connection was obtained from an
// 	// idle pool.
// 	IsConnWasIdle bool

// 	// ConnIdleTime is a duration how long the connection was previously
// 	// idle, if IsConnWasIdle is true.
// 	ConnIdleTime time.Duration

// 	// RequestAttempt is to represent the request attempt made during a Resty
// 	// request execution flow, including retry count.
// 	RequestAttempt int

// 	// RemoteAddr returns the remote network address.
// 	RemoteAddr net.Addr
// }

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// ClientTrace struct and its methods
//_______________________________________________________________________

// tracer struct maps the `httptrace.ClientTrace` hooks into Fields
// with same naming for easy understanding. Plus additional insights
// Request.
// type clientTrace struct {
// 	getConn              time.Time
// 	dnsStart             time.Time
// 	dnsDone              time.Time
// 	connectDone          time.Time
// 	tlsHandshakeStart    time.Time
// 	tlsHandshakeDone     time.Time
// 	gotConn              time.Time
// 	gotFirstResponseByte time.Time
// 	endTime              time.Time
// 	gotConnInfo          httptrace.GotConnInfo
// }

type clientTrace struct {
	getConn              int64
	dnsStart             int64
	dnsDone              int64
	connectDone          int64
	tlsHandshakeStart    int64
	tlsHandshakeDone     int64
	gotConn              int64
	gotFirstResponseByte int64
	addrs                []net.IPAddr
	endTime              time.Time
	gotConnInfo          httptrace.GotConnInfo
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Trace unexported methods
// with transport,can work with http.Client.Do(req) when client with transport
//_______________________________________________________________________

func (t *traceInfos) newTrace() *httptrace.ClientTrace {
	cliTrace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			t.dnsStart = time.Now().UnixMicro()
		},
		DNSDone: func(dnsinfo httptrace.DNSDoneInfo) {
			t.dnsDone = time.Now().UnixMicro()
			t.addrs = dnsinfo.Addrs
			// t.DNSLookup = t.dnsDone.Sub(t.dnsStart)
		},
		ConnectStart: func(_, _ string) {
			if t.dnsDone == 0 {
				t.dnsDone = time.Now().UnixMicro()
				time.Now().IsZero()
			}
			if t.dnsStart == 0 {
				t.dnsStart = t.dnsDone
			}
		},
		ConnectDone: func(net, addr string, err error) {
			t.connectDone = time.Now().UnixMicro()
		},
		GetConn: func(_ string) {
			t.getConn = time.Now().UnixMicro()
		},
		GotConn: func(ci httptrace.GotConnInfo) {
			t.gotConn = time.Now().UnixMicro()
			t.gotConnInfo = ci
			// t.ConnIdleTime = t.gotConnInfo.IdleTime
		},
		GotFirstResponseByte: func() {
			t.gotFirstResponseByte = time.Now().UnixMicro()
			// t.ServerTime = t.gotFirstResponseByte.Sub(t.gotConn)
		},
		TLSHandshakeStart: func() {
			t.tlsHandshakeStart = time.Now().UnixMicro()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			t.tlsHandshakeDone = time.Now().UnixMicro()
			// t.TLSHandshake = t.tlsHandshakeDone.Sub(t.tlsHandshakeStart)
		},
	}
	return cliTrace
}

func (t *traceInfos) createContextwithTransport() context.Context {
	return httptrace.WithClientTrace(
		context.Background(),
		t.newTrace(),
	)
}
