package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/nicumicle/parallel/internal/parallelhttp"
)

type (
	// RequestPayload handles the request payload for running parallel requests
	RequestPayload struct {
		Method         string            `json:"method"`
		Endpoint       string            `json:"endpoint"`
		Body           json.RawMessage   `json:"body"`
		Headers        map[string]string `json:"headers"`
		Parallel       int               `json:"parallel"`
		RequestTimeout int               `json:"request_timeout"`
		MaxDuration    int               `json:"max_duration"`
	}

	// Summary represents aggregate statistics
	Summary struct {
		TotalRequests int     `json:"total_requests"`
		SuccessCount  int     `json:"success_count"`
		ErrorCount    int     `json:"error_count"`
		AvgDuration   string  `json:"avg_duration"`
		Latency       Latency `json:"latency"`
	}

	Latency struct {
		P50 string `json:"p50"`
		P90 string `json:"p90"`
		P99 string `json:"p99"`
	}

	Result struct {
		Time       time.Time `json:"time"`
		StatusCode int       `json:"status_code"`
		Duration   string    `json:"duration"`
		Error      *string   `json:"error"`
	}

	// ResponsePayload is what the API returns
	ResponsePayload struct {
		Results []Result `json:"results"`
		Summary Summary  `json:"summary"`
	}
)

func HandlerParallel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		RespondError(w, ErrorMethodNotAllowed)

		return
	}

	var payload RequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondError(w, ErrorBadRequest)
		return
	}

	p := parallelhttp.New(time.Duration(payload.RequestTimeout) * time.Millisecond)

	result, err := p.Run(ctx, parallelhttp.Input{
		Method:   payload.Method,
		Endpoint: payload.Endpoint,
		Body:     payload.Body,
		Parallel: payload.Parallel,
		Headers:  payload.Headers,
		Duration: time.Duration(payload.MaxDuration) * time.Millisecond,
	})
	if err != nil {
		RespondError(w, Error{
			Code:       "validation.error",
			Title:      err.Error(),
			StatusCode: http.StatusUnprocessableEntity,
		})

		return
	}

	totalRequests := len(result.Requests)
	totalDuration := time.Duration(0)
	summary := Summary{
		TotalRequests: totalRequests,
		AvgDuration:   "0s",
		Latency: Latency{
			P50: result.Stats.Latency.P50,
			P90: result.Stats.Latency.P90,
			P99: result.Stats.Latency.P99,
		},
	}
	responseResults := make([]Result, totalRequests)
	for i, r := range result.Requests {
		// Calculate stats
		if r.Error == nil && r.Response != nil && r.Response.StatusCode < http.StatusBadRequest {
			summary.SuccessCount++
		} else {
			summary.ErrorCount++
		}

		res := Result{
			Error: r.ErrorMessage,
		}

		if r.Response != nil {
			totalDuration += r.Response.Duration

			res.Time = r.Response.Time
			res.StatusCode = r.Response.StatusCode
			res.Duration = r.Response.DurationH
		}

		responseResults[i] = res
	}

	// Calculate average
	if totalRequests > 0 {
		summary.AvgDuration = (totalDuration / time.Duration(totalRequests)).String()
	}

	RespondOK(w, ResponsePayload{
		Results: responseResults,
		Summary: summary,
	})
}
