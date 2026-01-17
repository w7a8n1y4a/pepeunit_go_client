package pepeunit

import "time"

// Version represents the library version
const Version = "1.1.1"

// DefaultCycleSpeed is the default cycle speed for the main loop
const DefaultCycleSpeed = 100 * time.Millisecond

// DefaultRestartMode is the default restart mode
const DefaultRestartMode = RestartModeRestartExec
