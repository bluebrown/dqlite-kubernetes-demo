package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

/*
	This program exists only for local development to provide a way
	for docker compose to perform health checks
*/

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: httpcheck [OPTIONS] URL\n")
		flag.PrintDefaults()
	}

	timeout := flag.Duration("timeout", time.Second, "timeout")
	method := flag.String("method", "GET", "method")
	minStatus := flag.Int("min-status", 200, "min status")
	maxStatus := flag.Int("max-status", 299, "max status")

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	if err := httpCheck(*timeout, *method, flag.Arg(0), *minStatus, *maxStatus); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

var ErrNotHealthy = fmt.Errorf("not healthy")

func httpCheck(timeout time.Duration, method string, url string, minStatus int, maxStatus int) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	res.Body.Close()
	sc := res.StatusCode
	if sc < minStatus || sc > maxStatus {
		return fmt.Errorf("bad status: status: %d: %w", sc, ErrNotHealthy)
	}
	return nil
}
