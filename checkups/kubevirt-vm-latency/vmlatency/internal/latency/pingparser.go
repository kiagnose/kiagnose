package latency

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/status"
)

func ParsePingResults(pingResult string) status.Results {
	var result status.Results
	latencyPattern := regexp.MustCompile(`(round-trip|rtt)\s+\S+\s*=\s*([0-9.]+)/([0-9.]+)/([0-9.]+)/([0-9.]+)\s*ms`)

	matches := latencyPattern.FindAllStringSubmatch(pingResult, -1)
	for _, item := range matches {
		min, err := time.ParseDuration(fmt.Sprintf("%sms", strings.TrimSpace(item[2])))
		if err != nil {
			log.Printf("failed to parse min latency from result: %v", err)
		}
		result.MinLatency = min

		average, err := time.ParseDuration(fmt.Sprintf("%sms", strings.TrimSpace(item[3])))
		if err != nil {
			log.Printf("failed to parse average jitter from result: %v", err)
		}
		result.AvgLatency = average

		max, err := time.ParseDuration(fmt.Sprintf("%sms", strings.TrimSpace(item[4])))
		if err != nil {
			log.Printf("failed to parse max latency from result: %v", err)
		}
		result.MaxLatency = max
	}

	return result
}
