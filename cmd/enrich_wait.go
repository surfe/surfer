package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Surfe/surfer/internal/client"
	"github.com/Surfe/surfer/pkg/output"
)

// enrichInProgressStatuses are the status values that mean "keep polling".
// Anything else (COMPLETED, FAILED, or an absent status) ends the wait.
var enrichInProgressStatuses = map[string]bool{
	"IN_PROGRESS": true,
	"PENDING":     true,
	"QUEUED":      true,
	"PROCESSING":  true,
}

// addWaitFlags registers the shared --wait/--poll-interval/--timeout flags on an
// enrich command.
func addWaitFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("wait", false, "Poll until enrichment completes and print the final result")
	cmd.Flags().Duration("poll-interval", 2*time.Second, "How often to poll while waiting (with --wait)")
	cmd.Flags().Duration("timeout", 2*time.Minute, "Max time to wait for completion (with --wait)")
}

// emitEnrichResult prints the start response, or — when --wait is set — polls the
// enrichment status endpoint until it finishes and prints the final result.
// statusBase is the collection status path, e.g. "/v2/companies/enrich".
func emitEnrichResult(cmd *cobra.Command, statusBase string, start map[string]any) error {
	wait, _ := cmd.Flags().GetBool("wait")
	if !wait {
		return output.Print(os.Stdout, start, outputFormat(cmd))
	}

	id, _ := start["enrichmentID"].(string)
	if id == "" {
		// Nothing to poll on — surface what the start call returned.
		return output.Print(os.Stdout, start, outputFormat(cmd))
	}

	interval, _ := cmd.Flags().GetDuration("poll-interval")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	result, err := waitForEnrichment(newAPIClient(), statusBase+"/"+id, interval, timeout)
	if err != nil {
		if result != nil {
			_ = output.Print(os.Stdout, result, outputFormat(cmd))
		}
		return err
	}
	return output.Print(os.Stdout, result, outputFormat(cmd))
}

// waitForEnrichment polls statusPath until the enrichment is no longer in progress
// or the timeout elapses. It shows a spinner on stderr when attached to a terminal.
func waitForEnrichment(c *client.Client, statusPath string, interval, timeout time.Duration) (any, error) {
	stop := startSpinner("Waiting for enrichment to complete")
	defer stop()

	deadline := time.Now().Add(timeout)
	for {
		var result map[string]any
		if err := c.Get(statusPath, &result); err != nil {
			return nil, err
		}
		status, _ := result["status"].(string)
		if !enrichInProgressStatuses[strings.ToUpper(status)] {
			return result, nil
		}
		if time.Now().After(deadline) {
			return result, fmt.Errorf("timed out after %s waiting for enrichment (last status: %s)", timeout, status)
		}
		time.Sleep(interval)
	}
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// startSpinner renders a spinner on stderr until the returned stop func is called.
// It is a no-op when stderr is not a terminal (piped/redirected), keeping stdout
// machine output clean. The stop func blocks until the spinner line is cleared.
func startSpinner(msg string) func() {
	if !isTerminal(os.Stderr) {
		return func() {}
	}

	return spinAsync(os.Stderr, msg, 120*time.Millisecond)
}

// spinAsync runs spin in a goroutine writing to w and returns a stop func that
// halts it and blocks until the line is cleared. Separated from startSpinner so
// the goroutine lifecycle is testable without a real terminal.
func spinAsync(w io.Writer, msg string, interval time.Duration) func() {
	done := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		defer close(finished)
		spin(w, msg, interval, done)
	}()

	return func() {
		close(done)
		<-finished
	}
}

// spin renders an animated spinner to w on each interval tick until done is
// closed, then clears the line. Extracted from startSpinner so it can be tested
// without a real terminal.
func spin(w io.Writer, msg string, interval time.Duration, done <-chan struct{}) {
	frames := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
	start := time.Now()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	i := 0
	for {
		select {
		case <-done:
			fmt.Fprint(w, "\r\033[K") // clear the spinner line
			return
		case <-ticker.C:
			fmt.Fprintf(w, "\r%c %s (%ds)", frames[i%len(frames)], msg, int(time.Since(start).Seconds()))
			i++
		}
	}
}
