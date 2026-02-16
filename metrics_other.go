//go:build !linux

package main

// readProcessRSS is a no-op on non-Linux platforms where /proc is unavailable.
func readProcessRSS() float64 { return 0 }

// readProcessCPUTicks is a no-op on non-Linux platforms where /proc is unavailable.
func readProcessCPUTicks() int64 { return 0 }
