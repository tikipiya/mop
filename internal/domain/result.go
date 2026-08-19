package domain

import "time"

type Status string

const (
	StatusUnknown  Status = "UNKNOWN"
	StatusChecking Status = "CHECKING"
	StatusOnline   Status = "ONLINE"
	StatusOffline  Status = "OFFLINE"
	StatusError    Status = "ERROR"
)

// Result contains only normalized data that is safe for presentation. Pointer
// fields distinguish a missing value from a real zero reported by the server.
type Result struct {
	Target          Target
	ResolvedTarget  *Target
	Status          Status
	Latency         *time.Duration
	VersionName     string
	Protocol        *int
	PlayersOnline   *int
	PlayersMax      *int
	MOTD            string
	CheckedAt       time.Time
	Warning         string
	ModInfoDetected bool
	ModLoader       string
	ModCount        *int
	ModInfoWarning  string
}
