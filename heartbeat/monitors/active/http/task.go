// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package http

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
	"github.com/elastic/beats/v7/heartbeat/monitors/active/dialchain/tlsmeta"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/look"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/active/dialchain"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/reason"
	"github.com/elastic/beats/v7/libbeat/beat"
)

type requestFactory func() (*http.Request, error)

type respResult struct {
	start      time.Time
	retrytimes int
	traceInfo  *traceInfos
	resp       *http.Response
	errReason  reason.Reason
}

func newHTTPMonitorHostJob(
	addr string,
	config *Config,
	transport http.RoundTripper,
	enc contentEncoder,
	body []byte,
	validator multiValidator,
) (jobs.Job, error) {

	var (
		reqFactory requestFactory = func() (*http.Request, error) { return buildRequest(addr, config, enc) }
		traceInfo                 = new(traceInfos)
	)
	return jobs.MakeSimpleJob(func(event *beat.Event) error {
		var redirects []string
		// by John
		// add async & sync
		if config.Sync.Enabled {
			config.Sync.Bid = config.ID + time.Now().Format("200601021504")
		}
		eventext.MergeEventFields(event, mapstr.M{"async": &config.Async, "sync": &config.Sync})
		client := &http.Client{
			// Trace visited URLs when redirects occur
			CheckRedirect: makeCheckRedirect(config.MaxRedirects, &redirects),
			Timeout:       config.Transport.Timeout,
			Transport:     transport,
		}

		req, err := reqFactory()
		if err != nil {
			return fmt.Errorf("could not make http request: %w", err)
		}

		_, err = execPing(event, client, req, body, traceInfo, &config.Check.Response, &config.Retry, config.Transport.Timeout, validator, config.Response)
		if len(redirects) > 0 {
			_, _ = event.PutValue("http.response.redirects", redirects)
		}
		return err
	}), nil
}

func newHTTPMonitorIPsJob(
	config *Config,
	addr string,
	tls *tlscommon.TLSConfig,
	enc contentEncoder,
	body []byte,
	validator multiValidator,
) (jobs.Job, error) {

	var reqFactory requestFactory = func() (*http.Request, error) { return buildRequest(addr, config, enc) }

	hostname, port, err := splitHostnamePort(addr)
	if err != nil {
		return nil, err
	}

	pingFactory := createPingFactory(config, port, tls, reqFactory, body, validator)
	job, err := monitors.MakeByHostJob(hostname, config.Mode, monitors.NewStdResolver(), pingFactory)

	return job, err
}

func createPingFactory(
	config *Config,
	port uint16,
	tls *tlscommon.TLSConfig,
	reqFactory requestFactory,
	body []byte,
	validator multiValidator,
) func(*net.IPAddr) jobs.Job {
	timeout := config.Transport.Timeout

	return monitors.MakePingIPFactory(func(event *beat.Event, ip *net.IPAddr) error {
		req, err := reqFactory()
		if err != nil {
			return fmt.Errorf("could not create http request: %w", err)
		}
		isTLS := req.URL.Scheme == "https"

		addr := net.JoinHostPort(ip.String(), strconv.Itoa(int(port)))
		d := &dialchain.DialerChain{
			Net: dialchain.MakeConstAddrDialer(addr, dialchain.TCPDialer(timeout)),
		}

		if isTLS {
			d.AddLayer(dialchain.TLSLayer(tls, timeout))
		}

		// dialer, err := d.Build(event)
		if err != nil {
			return err
		}

		var (
			writeStart, readStart, writeEnd time.Time
			traceInfo                       = new(traceInfos)
		)
		// Ensure memory consistency for these callbacks.
		// It seems they can be invoked still sometime after the request is done
		cbMutex := sync.Mutex{}

		// We don't support redirects for IP jobs, so this effectively just
		// prevents following redirects in this case, we know that
		// config.MaxRedirects must be zero to even be here
		checkRedirect := makeCheckRedirect(0, nil)
		// transport := &SimpleTransport{
		//	Dialer: dialer,
		// 	OnStartWrite: func() {
		// 		cbMutex.Lock()
		// 		writeStart = time.Now()
		// 		cbMutex.Unlock()
		// 	},
		// 	OnEndWrite: func() {
		// 		cbMutex.Lock()
		// 		writeEnd = time.Now()
		// 		cbMutex.Unlock()
		// 	},
		// 	OnStartRead: func() {
		// 		cbMutex.Lock()
		// 		readStart = time.Now()
		// 		cbMutex.Unlock()
		// 	},
		// }
		client := &http.Client{
			CheckRedirect: checkRedirect,
			Timeout:       timeout,
			// Transport:     httpcommon.HeaderRoundTripper(transport, map[string]string{"User-Agent": userAgent}),
		}

		end, err := execPing(event, client, req, body, traceInfo, &config.Check.Response, &config.Retry, timeout, validator, config.Response)
		cbMutex.Lock()
		defer cbMutex.Unlock()

		if !readStart.IsZero() {
			eventext.MergeEventFields(event, mapstr.M{
				"http": mapstr.M{
					"rtt": mapstr.M{
						"write_request":   look.RTT(writeEnd.Sub(writeStart)),
						"response_header": look.RTT(readStart.Sub(writeStart)),
					},
				},
			})
		}
		if !writeStart.IsZero() {
			_, _ = event.PutValue("http.rtt.validate", look.RTT(end.Sub(writeStart)))
			_, _ = event.PutValue("http.rtt.content", look.RTT(end.Sub(readStart)))
		}

		return err
	})
}

func buildRequest(addr string, config *Config, enc contentEncoder) (*http.Request, error) {
	method := strings.ToUpper(config.Check.Request.Method)
	request, err := http.NewRequestWithContext(context.TODO(), method, addr, nil)
	if err != nil {
		return nil, err
	}
	request.Close = true

	if config.Username != "" {
		request.SetBasicAuth(config.Username, config.Password)
	}
	for k, v := range config.Check.Request.SendHeaders {
		// defining the Host header isn't enough. See https://github.com/golang/go/issues/7682
		if k == "Host" {
			request.Host = v
		}

		request.Header.Add(k, v)
	}

	if enc != nil {
		enc.AddHeaders(&request.Header)
	}

	return request, nil
}

func execPing(
	event *beat.Event,
	client *http.Client,
	req *http.Request,
	reqBody []byte,
	traceInfo *traceInfos,
	respCfg *responseParameters,
	retry *retryConfig, //by John
	timeout time.Duration,
	validator multiValidator,
	responseConfig responseConfig,
) (end time.Time, err error) {
	// // ctx, cancel := context.WithTimeout(context.Background(), timeout+time.Duration(float64(retry.Retries)*float64(retry.WaitTime)))
	// ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// defer cancel()

	// req = attachRequestBody(&ctx, req, reqBody)

	// // Send the HTTP request. We don't immediately return on error since
	// // we may want to add additional fields to contextualize the error.
	// start, resp, errReason := execRequest(client, req)

	// by John
	// retry function
	var (
		start, rstart time.Time
		resp          *http.Response
		errReason     reason.Reason
		retrytimes    int

		// traceInfo  *traceInfos
	)
	start = time.Now()

	// if retry.Retries == 0 {
	// 	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// 	defer cancel()
	// 	dnserr := dnstrace(req, traceInfo)
	// 	req = attachRequestBody(&ctx, req, reqBody)
	// 	rstart, traceInfo, resp, errReason = execRequest(client, traceInfo, req)
	// 	if dnserr != nil {
	// 		errReason = reason.IOFailed(dnserr)
	// 	}
	// } else if retry.Retries > 0 {
	// 	for i := 0; i <= retry.Retries; i++ {
	// 		ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// 		dnserr := dnstrace(req, traceInfo)
	// 		req = attachRequestBody(&ctx, req, reqBody)
	// 		// excute the request backoff retries with increasing timeout duration up until X amount of retries.
	// 		rstart, traceInfo, resp, errReason = execRequest(client, traceInfo, req)
	// 		if dnserr != nil {
	// 			errReason = reason.IOFailed(dnserr)
	// 		}
	// 		retrytimes = i
	// 		cancel() //By John: Dont use defer in `for` loop，context deadline exceeded will export.
	// 		if resp != nil && errReason == nil {
	// 			break
	// 		}
	// 		if i != retry.Retries {
	// 			time.Sleep(retry.WaitTime)
	// 		}
	// 	}
	// }

	// 创建resultChan用于接收结果
	resultChan := make(chan *respResult)

	// 启动sendHTTPRequest的goroutine执行HTTP请求
	go sendHTTPRequest(resultChan, timeout, respCfg, retry, req, traceInfo, reqBody, client)

	// 使用select语句从resultChan中读取数据
	select {
	case result := <-resultChan:
		if result != nil {
			rstart = result.start
			traceInfo = result.traceInfo
			retrytimes = result.retrytimes
			resp = result.resp
			errReason = result.errReason
		}
	}
	// by John
	// add retry_times
	eventext.MergeEventFields(event, mapstr.M{"http": mapstr.M{
		"trace": mapstr.M{
			"addrs":      traceInfo.addrs,
			"start_time": start,
			"retries":    retrytimes,
		}}})

	// If we have no response object or an error was set there probably was an IO error, we can skip the rest of the logic
	// since that logic is for adding metadata relating to completed HTTP transactions that have errored
	// in other ways
	if resp == nil || errReason != nil {
		var ecsErr *ecserr.ECSErr
		var urlError *url.Error
		// by John
		// add end time
		eventext.MergeEventFields(event, mapstr.M{"http": mapstr.M{"trace": mapstr.M{"end_time": time.Now()}}})
		if errors.As(errReason.Unwrap(), &ecsErr) {
			return time.Now(), ecsErr
		} else if errors.As(errReason.Unwrap(), &urlError) {
			var certErr x509.CertificateInvalidError
			if errors.As(urlError, &certErr) {
				tlsFields := tlsmeta.CertFields(certErr.Cert, nil)
				event.Fields.DeepUpdate(mapstr.M{"tls": tlsFields})

			}
		}
		return time.Now(), errReason
	}

	bodyFields, mimeType, errReason := processBody(resp, responseConfig, validator)

	responseFields := mapstr.M{
		"status_code": resp.StatusCode,
		"body":        bodyFields,
	}

	if mimeType != "" {
		responseFields["mime_type"] = mimeType
	}

	if responseConfig.IncludeHeaders {
		headerFields := mapstr.M{}
		for canonicalHeaderKey, vals := range resp.Header {
			if len(vals) > 1 {
				headerFields[canonicalHeaderKey] = vals
			} else {
				headerFields[canonicalHeaderKey] = vals[0]
			}
		}
		responseFields["headers"] = headerFields
	}

	httpFields := mapstr.M{"response": responseFields}

	eventext.MergeEventFields(event, mapstr.M{"http": httpFields})

	// Mark the end time as now, since we've finished downloading
	end = time.Now()

	// by John
	// add trace fields
	// See 'heartbeat/monitors/active/http/trace.go' for definitions of all fields
	traceInfo.endTime = end
	// traceDuration := mapstr.M{
	// 	"dnslookup":    traceInfo.DNSLookup,
	// 	"tlshandshake": traceInfo.TLSHandshake,
	// 	"servetime":    traceInfo.ServerTime,
	// 	"connIdletime": traceInfo.ConnIdleTime,
	// }
	eventext.MergeEventFields(event, mapstr.M{"http": mapstr.M{
		"trace": mapstr.M{
			"get_conn":                traceInfo.getConn,
			"dns_start":               traceInfo.dnsStart,
			"dns_done":                traceInfo.dnsDone,
			"tls_handshake_start":     traceInfo.tlsHandshakeStart,
			"tls_handshake_done":      traceInfo.tlsHandshakeDone,
			"got_conn":                traceInfo.gotConn,
			"got_first_Response_byte": traceInfo.gotFirstResponseByte,
			"end_time":                traceInfo.endTime,
			// "connect_done":            traceInfo.connectDone,
			// "trace_duration":          traceDuration,
		},
	}})

	// Enrich event with TLS information when available. This is useful when connecting to an HTTPS server through
	// a proxy.
	if resp.TLS != nil {
		tlsFields := mapstr.M{}
		tlsmeta.AddTLSMetadata(tlsFields, *resp.TLS, tlsmeta.UnknownTLSHandshakeDuration)
		eventext.MergeEventFields(event, tlsFields)
	}

	// Add total HTTP RTT
	eventext.MergeEventFields(event, mapstr.M{"http": mapstr.M{
		"rtt": mapstr.M{
			"total": look.RTT(end.Sub(rstart)),
		},
	}})

	return end, errReason
}

func attachRequestBody(ctx *context.Context, req *http.Request, body []byte) *http.Request {
	req = req.WithContext(*ctx)
	if len(body) > 0 {
		req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		req.ContentLength = int64(len(body))
	}

	return req
}

// execute the request. Note that this does not close the resp body, which should be done by caller
func execRequest(client *http.Client, traceInfo *traceInfos, req *http.Request) (start time.Time, traceinfo *traceInfos, resp *http.Response, errReason reason.Reason) {
	//add httpclinenttrace infomation
	// ctx := traceInfo.createContext(req.Context())
	ctx := traceInfo.createContextwithTransport()
	req = req.WithContext(ctx)
	start = time.Now()
	// client = new(http.Client)
	resp, err := client.Do(req)
	// resp, err := http.DefaultTransport.RoundTrip(req)

	// Since the HTTP client is very old we can't use errors.Is, but must
	// use this ancient bit of cruft to determine if we couldn't connect
	// The nomenclature about this being a timeout is actually wrong
	// this happens on all sorts of connection errors, so it's double lame
	if os.IsTimeout(err) {
		err = ecserr.NewCouldNotConnectErr(req.URL.Hostname(), req.URL.Port(), err)
	}

	if err != nil {
		// by john
		// add ECSError Information.
		if strings.Contains(err.Error(), "connection refused") {
			err = ecserr.NewConnectFailedErr(err)
		} else if strings.Contains(err.Error(), "no route to host") {
			err = ecserr.NewRouteFailedErr(err)
		} else if strings.Contains(err.Error(), "connection reset by peer") || strings.Contains(err.Error(), "EOF") {
			err = ecserr.NewNetWorkFailedErr(err)
		}
		return start, traceInfo, nil, reason.IOFailed(err)
	}

	return start, traceInfo, resp, nil
}

func splitHostnamePort(addr string) (string, uint16, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return "", 0, err
	}
	host := u.Host
	// Try to add a default port if needed
	if strings.LastIndex(host, ":") == -1 {
		switch u.Scheme {
		case urlSchemaHTTP:
			host += ":80"
		case urlSchemaHTTPS:
			host += ":443"
		}
	}
	host, port, err := net.SplitHostPort(host)
	if err != nil {
		return "", 0, err
	}
	p, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return "", 0, fmt.Errorf("'%v' is no valid port number in '%v'", port, u.Host)
	}
	return host, uint16(p), nil
}

// makeCheckRedirect checks if max redirects are exceeded, also append to the redirects list if we're tracking those.
// It's kind of ugly to return a result via a pointer argument, but it's the interface the
// golang HTTP client gives us.
func makeCheckRedirect(max int, redirects *[]string) func(*http.Request, []*http.Request) error {
	if max == 0 {
		return func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return func(r *http.Request, via []*http.Request) error {
		if redirects != nil {
			*redirects = append(*redirects, r.URL.String())
		}

		if max == len(via) {
			return http.ErrUseLastResponse
		}
		return nil
	}
}

func dnstrace(req *http.Request, traceInfo *traceInfos) error {
	// by john
	// add dnstrace info;
	clitrace := traceInfo.newTrace()
	// 手动触发httptrace的DNSStart钩子函数
	host := req.Host
	if strings.Contains(host, ":") {
		host, _, _ = net.SplitHostPort(req.Host)
	}
	clitrace.DNSStart(httptrace.DNSStartInfo{Host: req.URL.Host})
	addrs, lerr := net.LookupHost(host)
	if lerr != nil {
		return ecserr.NewDNSLookupFailedErr(req.Host, lerr)
	}
	// 转换成 []net.IPAddr 类型
	ipAddrs := make([]net.IPAddr, len(addrs))
	for i, addr := range addrs {
		ipAddrs[i] = net.IPAddr{IP: net.ParseIP(addr)}
	}
	// 手动触发httptrace的DNSDone钩子函数
	clitrace.DNSDone(httptrace.DNSDoneInfo{Addrs: ipAddrs})
	// 手动触发httptrace的ConnectStart和ConnectDone钩子函数
	if addrs != nil {
		clitrace.ConnectDone("tcp", addrs[0], nil)
	}
	return nil
}

// 发送HTTP请求
func sendHTTPRequest(resultChan chan<- *respResult, timeout time.Duration, config *responseParameters, retry *retryConfig, req *http.Request, traceInfo *traceInfos, reqBody []byte, client *http.Client) {
	var (
		result     *respResult
		errReason  reason.Reason
		retrytimes int
		resp       *http.Response
		rstart     time.Time
	)
	// 使用Context进行HTTP请求
	for retries := 0; retries <= retry.Retries; retries++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		dnserr := dnstrace(req, traceInfo)
		req = attachRequestBody(&ctx, req, reqBody)
		// excute the request backoff retries with increasing timeout duration up until X amount of retries.
		rstart, traceInfo, resp, errReason = execRequest(client, traceInfo, req)
		if dnserr != nil {
			errReason = reason.IOFailed(dnserr)
		}
		retrytimes = retries

		cancel() //By John: Dont use defer in `for` loop，context deadline exceeded will export.
		result = &respResult{
			start:      rstart,
			retrytimes: retrytimes,
			traceInfo:  traceInfo,
			resp:       resp,
			errReason:  errReason,
		}

		// 异常状态码返回时errReason有为空的情况，需要具体判断状态码；
		// // TODO: 涉及自定义状态码的情况需要解决。
		// 处理自定义状态码的黑白名单；
		var code_err error
		if resp != nil {
			if len(config.Status) > 0 {
				code_err = checkStausCodes(resp, config.Status)
			} else if len(config.BadStatus) > 0 {
				code_err = checkBadStausCodes(resp, config.BadStatus)
			} else {
				code_err = checkStatusOK(resp)
			}
		}

		if resp != nil && errReason == nil && code_err == nil {
			resultChan <- result
			return
		}

		if retries != retry.Retries {
			time.Sleep(retry.WaitTime)
		}
	}

	// 最大重试次数达到后，发送响应结果到Channel，并报告错误

	resultChan <- result
}
