package parallelhttp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

var errCanceled = errors.New("canceled")

type ParallelHTTP struct {
	client *http.Client
}

type (
	// Result represents the outcome of a single HTTP request.
	Result struct {
		Requests []Call `json:"requests" yaml:"requests"`
		Stats    Stats  `json:"stats" yaml:"stats"`
	}

	Stats struct {
		StartTime time.Time `json:"start_time" yaml:"start_time"`
		EndTime   time.Time `json:"end_time" yaml:"end_time"`
		Duration  string    `json:"duration" yaml:"duration"`
		Latency   Latency   `json:"latency" yaml:"latency"`
	}
	Latency struct {
		P50 string `json:"p50" yaml:"p50"`
		P90 string `json:"p90" yaml:"p90"`
		P99 string `json:"p99" yaml:"p99"`
	}

	Call struct {
		Response     *Response `json:"response" yaml:"response"`
		Error        error     `json:"error" yaml:"error"`
		ErrorMessage *string   `json:"error_message" yaml:"error_message"`
	}
	Response struct {
		StatusCode int           `json:"status_code" yaml:"status_code"`
		Time       time.Time     `json:"time" yaml:"time"`
		Duration   time.Duration `json:"duration" yaml:"duration"`
		DurationH  string        `json:"duration_h" yaml:"duration_h"`
	}
)

func (c *Call) SetError(err error) {
	c.Error = err
	if err == nil {
		return
	}
	msg := err.Error()
	c.ErrorMessage = &msg
}

func (r *Response) SetDuration(d time.Duration) {
	r.Duration = d
	r.DurationH = d.String()
}

type Input struct {
	Method   string
	Endpoint string
	Body     []byte
	Headers  map[string]string
	Parallel int
	Duration time.Duration
}

func (i *Input) Validate() error {
	switch {
	case i.Endpoint == "":
		return errors.New("endpoint is required")
	case !slices.Contains([]string{"GET", "POST", "PUT", "PATCH", "DELETE"}, strings.ToUpper(i.Method)):
		return errors.New("invalid value for method")
	case i.Parallel <= 0:
		return errors.New("invalid value for parallel")
	case i.Duration < 0*time.Second:
		return errors.New("invalid value for duration")
	default:
		return nil
	}
}

func New(timeout time.Duration) *ParallelHTTP {
	return &ParallelHTTP{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Run performs parallel HTTP requests to the given endpoint.
func (p *ParallelHTTP) Run(ctx context.Context, input Input) (Result, error) {
	result := Result{
		Requests: make([]Call, 0),
		Stats: Stats{
			StartTime: time.Now(),
		},
	}
	if p.client.Timeout < 0 {
		return result, errors.New("invalid value for timeout")
	}
	if err := input.Validate(); err != nil {
		return result, err
	}
	var wg sync.WaitGroup
	calls := make(chan Call, input.Parallel)

	for i := 0; i < input.Parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			calls <- p.process(ctx, input)
		}()
	}

	var wgEnd sync.WaitGroup
	wgEnd.Add(1)
	go func() {
		defer wgEnd.Done()

		wg.Wait()
		result.Stats.EndTime = time.Now()

		close(calls)
	}()

	wgEnd.Wait()

	for c := range calls {
		result.Requests = append(result.Requests, c)
	}

	result.Stats.Duration = result.Stats.EndTime.Sub(result.Stats.StartTime).String()
	result.Stats.Latency = latency(result.Requests)

	return result, nil
}

func (p *ParallelHTTP) process(ctx context.Context, input Input) Call {
	var cancel func()
	if input.Duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, input.Duration)
		defer cancel()
	}

	reqBody := io.Reader(nil)
	if input.Body != nil {
		reqBody = bytes.NewReader(input.Body)
	}

	req, err := http.NewRequestWithContext(ctx, input.Method, input.Endpoint, reqBody)
	if err != nil {
		msg := err.Error()
		return Call{
			Error:        err,
			ErrorMessage: &msg,
		}
	}
	if input.Headers != nil {
		for key, value := range input.Headers {
			req.Header.Set(key, value)
		}
	}

	call := Call{
		Response: &Response{
			Time: time.Now(),
		},
	}
	resp, err := p.client.Do(req)
	duration := time.Since(call.Response.Time)
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded),
		errors.Is(err, context.Canceled):
		call.SetError(errCanceled)

		return call
	case err != nil:
		parsedErr := parseError(err)
		call.SetError(parsedErr)

		return call
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	call.Response.SetDuration(duration)
	call.Response.StatusCode = resp.StatusCode

	return call
}

func parseError(err error) error {
	var netErr net.Error
	if errors.As(err, &netErr) {
		if strings.Contains(err.Error(), "no such host") {
			return errors.New("host not found")
		}
		if strings.Contains(err.Error(), "connection refused") {
			return errors.New("connection refused. The server may be down")
		}
		if netErr.Timeout() {
			return errors.New("request timed out")
		}
	}

	return err
}

func latency(calls []Call) Latency {
	data := make([]time.Duration, 0, len(calls))
	for _, c := range calls {
		if c.Error != nil || c.Response == nil || c.Response.Duration == 0 {
			continue
		}
		data = append(data, c.Response.Duration)
	}

	slices.Sort(data)

	n := len(data)
	if n == 0 {
		return Latency{
			P50: "0s",
			P90: "0s",
			P99: "0s",
		}
	}

	// helper to compute percentile index
	percentile := func(p float64) time.Duration {
		idx := max(int(math.Ceil(p*float64(n)))-1, 0)
		if idx >= n {
			idx = n - 1
		}
		return data[idx]
	}

	return Latency{
		P50: percentile(0.50).String(),
		P90: percentile(0.90).String(),
		P99: percentile(0.99).String(),
	}
}
