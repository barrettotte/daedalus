//go:build !linux

package daedalus

// ReadProcessRSS is a no-op on non-Linux platforms where /proc is unavailable.
func ReadProcessRSS() float64 { return 0 }

// ReadProcessCPUTicks is a no-op on non-Linux platforms where /proc is unavailable.
func ReadProcessCPUTicks() int64 { return 0 }
