package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/nicumicle/parallel/internal/parallelhttp"
)

func main() {
	var format string

	// Init flags
	method := flag.String("method", "GET", "Request Method. Default GET.")
	endpoint := flag.String("endpoint", "", "Request endpoint to be called.")
	parallel := flag.Int("parallel", 1, "Number of parallel calls. Default 1.")
	duration := flag.Duration("duration", 0, "Max duration for all calls. Example: 0->no limit, 1ms, 1s, 10m")
	timeout := flag.Duration("timeout", 0*time.Second, "Request timeout. Default 10s")
	flag.StringVar(&format, "format", "json", "Response format. One of: text, yaml, json. Default json.")
	flag.Parse()

	input := parallelhttp.Input{
		Method:   strings.ToUpper(*method),
		Endpoint: *endpoint,
		Parallel: *parallel,
		Duration: *duration,
	}
	p := parallelhttp.New(*timeout)

	r, err := p.Run(context.Background(), input)
	if err != nil {
		log.Fatalf("[ERROR]: %s\n", err.Error())
	}

	switch format {
	case "yaml":

		resultYaml, err := yaml.Marshal(r)
		if err != nil {
			log.Fatalf("marshal error: %s", err.Error())
		}

		fmt.Println(string(resultYaml))
	case "text":
		fmt.Println("Results:")
		fmt.Printf("\r %3s. %20s %25s %20s %15s \n ", "#", "Time", "Status Code", "Duration", "Error")
		for i, result := range r.Requests {
			statusCode := "-"
			duration := "-"
			callTime := "-"
			errorMessage := ""
			if result.Response != nil {
				statusCode = fmt.Sprintf("%d", result.Response.StatusCode)
				duration = result.Response.Duration.String()
				callTime = result.Response.Time.Format(time.RFC3339Nano)
			}
			if result.ErrorMessage != nil {
				errorMessage = *result.ErrorMessage
				statusCode = "-"
				duration = "-"
			}
			fmt.Printf("\r %3.d. %35s %6s %25s %15s \n ", i+1, callTime, statusCode, duration, errorMessage)
		}
		fmt.Println()
		fmt.Println("Stats:")
		fmt.Println(" ", "Start Time:", r.Stats.StartTime.Format(time.RFC3339Nano))
		fmt.Println(" ", "End Time:", r.Stats.EndTime.Format(time.RFC3339Nano))
		fmt.Println(" ", "Total Duration: ", r.Stats.Duration)
		fmt.Println("Latency:")
		fmt.Println(" ", "P50", r.Stats.Latency.P50)
		fmt.Println(" ", "P90", r.Stats.Latency.P90)
		fmt.Println(" ", "P99", r.Stats.Latency.P99)
	default:
		result, err := json.Marshal(r)
		if err != nil {
			log.Fatalf("marshal error: %s", err.Error())
		}

		fmt.Println(string(result))
	}
}
