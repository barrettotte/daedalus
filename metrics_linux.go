//go:build linux

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// readProcessRSS reads the resident set size from /proc/self/statm in megabytes.
func readProcessRSS() float64 {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}
	var size, resident int64
	if _, err := fmt.Sscanf(string(data), "%d %d", &size, &resident); err != nil {
		return 0
	}
	return float64(resident*int64(os.Getpagesize())) / 1024 / 1024
}

// readProcessCPUTicks reads utime + stime from /proc/self/stat in clock ticks.
func readProcessCPUTicks() int64 {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0
	}
	s := string(data)

	// Skip past comm field (may contain spaces/parens) by finding last ")"
	i := strings.LastIndex(s, ")")
	if i < 0 || i+2 >= len(s) {
		return 0
	}

	fields := strings.Fields(s[i+2:])
	if len(fields) < 13 {
		return 0
	}
	utime, _ := strconv.ParseInt(fields[11], 10, 64)
	stime, _ := strconv.ParseInt(fields[12], 10, 64)
	return utime + stime
}
