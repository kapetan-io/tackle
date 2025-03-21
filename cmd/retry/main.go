package main

import (
	"flag"
	"fmt"
	"github.com/kapetan-io/tackle/retry"
	"math/rand"
	"os"
	"path"
	"time"
)

// main outputs the following when called with no arguments
//
//  $ retry
//  Usage: retry -attempts 10 -min 500ms -max 1m0s -factor 1.5 -jitter 0.5
//
//  Attempt: 0 BackOff: 500ms WithJitter: 359.736082ms Jitter Range: [250ms - 750ms]
//  Attempt: 1 BackOff: 750ms WithJitter: 675.611954ms Jitter Range: [375ms - 1.125s]
//  Attempt: 2 BackOff: 1.125s WithJitter: 611.039003ms Jitter Range: [562.5ms - 1.6875s]
//  Attempt: 3 BackOff: 1.6875s WithJitter: 1.955070657s Jitter Range: [843.75ms - 2.53125s]
//  Attempt: 4 BackOff: 2.53125s WithJitter: 2.561133823s Jitter Range: [1.265625s - 3.796875s]
//  Attempt: 5 BackOff: 3.796875s WithJitter: 5.508541149s Jitter Range: [1.8984375s - 5.6953125s]
//  Attempt: 6 BackOff: 5.6953125s WithJitter: 4.373686656s Jitter Range: [2.84765625s - 8.54296875s]
//  Attempt: 7 BackOff: 8.54296875s WithJitter: 6.257106109s Jitter Range: [4.271484375s - 12.814453125s]
//  Attempt: 8 BackOff: 12.814453125s WithJitter: 12.994089755s Jitter Range: [6.407226562s - 19.221679687s]
//  Attempt: 9 BackOff: 19.221679687s WithJitter: 13.437161293s Jitter Range: [9.610839843s - 28.83251953s]
func main() {
	// Define command-line flags
	minDuration := flag.Duration("min", 500*time.Millisecond, "Minimum duration (e.g., 500ms, 1s, 1m)")
	maxDuration := flag.Duration("max", time.Minute, "Maximum duration (e.g., 1s, 1m, 1h)")
	factor := flag.Float64("factor", 1.5, "Factor to increase the duration")
	jitter := flag.Float64("jitter", 0.5, "Jitter value (between 0 and 1)")
	attempts := flag.Int("attempts", 10, "The number of attempts to simulate")
	help := flag.Bool("help", false, "Print help")
	flag.Parse()

	if *help {
		usage()
	}

	fmt.Printf("\nUsage: %s -attempts %d -min %v -max %v -factor %v -jitter %v\n\n", path.Base(os.Args[0]),
		*attempts, *minDuration, *maxDuration, *factor, *jitter)
	flag.Parse()

	r := retry.IntervalBackOff{
		Rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
		Min:    *minDuration,
		Max:    *maxDuration,
		Factor: *factor,
		Jitter: *jitter,
	}

	for i := 0; i < *attempts; i++ {
		fmt.Printf("%s\n", r.ExplainString(i))
	}
}

func usage() {
	fmt.Printf("Usage: %s [options]\n\n", path.Base(os.Args[0]))
	fmt.Println("This tool outputs back off times using retry.IntervalBackOff with user-specified values.")
	fmt.Println("Users can use this tool to fine tune backoff settings before using them in production")
	fmt.Println("\nOptions:")
	fmt.Println("  -min duration    Minimum duration (default: 500ms)")
	fmt.Println("                   Examples: 100ms, 1s, 500ms")
	fmt.Println("  -max duration    Maximum duration (default: 1m)")
	fmt.Println("                   Examples: 5s, 1m, 1h")
	fmt.Println("  -factor float    Factor to increase the duration (default: 1.5)")
	fmt.Println("                   Examples: 1.5, 2.0, 3.0")
	fmt.Println("  -jitter float    Jitter value between 0 and 1 (default: 0.5)")
	fmt.Println("                   Examples: 0.1, 0.5, 0.9")
	fmt.Println("  -attempts int    The number of attempts to simulate (default: 10)")
	fmt.Println("                   Examples: 10, 20, 50")
	fmt.Println("  -help bool       Output this usage information")
	fmt.Println("\nExamples:")
	fmt.Printf("  %s -attempts 10 -min 1s -max 2m -factor 2.0 -jitter 0.3\n", path.Base(os.Args[0]))
	fmt.Printf("  %s -min 200ms -max 30s -factor 1.8 -jitter 0.4\n", path.Base(os.Args[0]))
	os.Exit(0)
}
