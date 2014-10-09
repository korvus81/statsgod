/**
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package generator is a sample metric generator app.
package main

/**
 * Test generator used to send data to the statsgod process.
 */

import (
	"flag"
	"fmt"
	"github.com/jmcvetta/randutil"
	"math/rand"
	"net"
	"time"
)

var statsHost = flag.String("statsHost", "localhost", "Stats Hostname")
var statsPort = flag.Int("statsPort", 8125, "Stats Port")
var numMetrics = flag.Int("numMetrics", 10, "Number of metrics")
var flushTime = flag.Duration("flushTime", 2000*time.Millisecond, "Flush time")
var sleepTime = flag.Duration("sleepTime", 10*time.Nanosecond, "Sleep time")

const (
	// AvailableMemory is amount of available memory for the process.
	AvailableMemory = 10 << 20 // 10 MB, for example
	// AverageMemoryPerRequest is how much memory we want to use per request.
	AverageMemoryPerRequest = 10 << 10 // 10 KB
	// MAXREQS is how many requests.
	MAXREQS = AvailableMemory / AverageMemoryPerRequest
)

var statsPipeline = make(chan Metric, MAXREQS)

// Metric is our main data type.
type Metric struct {
	key        string // Name of the metric.
	metricType string // What type of metric is it (gauge, counter, timer)
}

var store []Metric

func main() {
	// Load command line options.
	flag.Parse()

	logger(fmt.Sprintf("Creating %d metrics", *numMetrics))

	store = generateMetricNames(*numMetrics)
	fmt.Println("Our new store:")
	fmt.Println(store)

	// Every X seconds we want to flush the metrics
	go loadTestMetrics(store)

	// Constantly process background Stats queue.
	go handleStatsQueue()

	select {} // block forever

}

func loadTestMetrics(store []Metric) {
	flushTicker := time.Tick(*flushTime)
	fmt.Printf("Flushing every %v\n", *flushTime)

	for {
		select {
		case <-flushTicker:
			fmt.Println("Tick...")
			for _, metric := range store {
				statsPipeline <- metric
			}
		}
	}
}

func handleStatsQueue() {
	for {
		metric := <-statsPipeline
		go sendMetricToStats(metric)
	}
}

func generateMetricNames(numMetrics int) []Metric {
	metricTypes := []string{
		"c",
		"g",
		"ms",
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < numMetrics; i++ {
		newMetricName, _ := randutil.String(20, randutil.Alphabet)
		newMetricNS := fmt.Sprintf("statsgod.test.%s", newMetricName)
		store = append(store, Metric{key: newMetricNS, metricType: metricTypes[r.Intn(len(metricTypes))]})
	}

	return store
}

// sendSingleMetricToGraphite formats a message and a value and time and sends to Graphite.
func sendMetricToStats(metric Metric) {
	var payload string
	fmt.Printf("Sending metric %s.%s to stats\n", metric.metricType, metric.key)

	c, err := net.Dial("tcp", fmt.Sprintf("%s:%d", *statsHost, *statsPort))
	if err != nil {
		fmt.Println("Could not connect to remote stats server")
		return
	}

	defer c.Close()

	rand.Seed(time.Now().UnixNano())

	if metric.metricType == "ms" {
		payload = fmt.Sprintf("%s:%d|ms", metric.key, rand.Intn(1000))
	} else if metric.metricType == "c" {
		payload = fmt.Sprintf("%s:1|c", metric.key)
	} else {
		payload = fmt.Sprintf("%s:%d|g", metric.key, rand.Intn(100))
	}

	//sv := strconv.FormatFloat(float64(v), 'f', 6, 32)
	//payload := fmt.Sprintf("%s %s %s", key, sv, t)
	//Trace.Printf("Payload: %v", payload)

	// Send to the connection
	fmt.Fprintf(c, payload)
}

func logger(msg string) {
	fmt.Println(msg)
}
