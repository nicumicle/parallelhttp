package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/nicumicle/parallel/internal/api"
	"github.com/nicumicle/parallel/internal/parallelhttp"
)

func main() {
	var format string

	// Init flags
	method := flag.String("method", "GET", "Request Method. Default GET.")
	endpoint := flag.String("endpoint", "", "Request endpoint to be called.")
	parallel := flag.Int("parallel", 1, "Number of parallel calls. Default 1.")
	duration := flag.Duration("duration", 0, "Max duration for all calls. Example: 0->no limit, 1ms, 1s, 10m. Default 0.")
	timeout := flag.Duration("timeout", 0*time.Second, "Request timeout. Default 10s")
	serve := flag.Bool("serve", false, "Starts the HTTP server.")
	port := flag.Int("port", 8080, "HTTP server port. Default 8080.")
	flag.StringVar(&format, "format", "json", "Response format. One of: text, yaml, json. Default json.")
	flag.Parse()

	input := parallelhttp.Input{
		Method:   strings.ToUpper(*method),
		Endpoint: *endpoint,
		Parallel: *parallel,
		Duration: *duration,
	}

	// HTTP Server
	if serve != nil && *serve {
		if err := runHTTP(*port); err != nil {
			log.Fatalf("Error: %s", err.Error())
		}

		return
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

func runHTTP(port int) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		BaseContext:  func(l net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,
		Handler:      api.NewAPI(),
	}

	srvErr := make(chan error, 1)
	go func() {
		log.Printf("Server started at: %d\n", port)
		srvErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-srvErr:
		return err
	case <-ctx.Done():
		stop()
	}

	return srv.Shutdown(context.Background())
}
