package daedalus

// ClockTicksPerSec is the assumed clock tick rate for CPU time calculations.
// On Linux this is USER_HZ (100). On other platforms the CPU metrics return 0,
// so the value is unused but must exist for compilation.
const ClockTicksPerSec = 100
