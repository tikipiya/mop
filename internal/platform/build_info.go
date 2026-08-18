package platform

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

func CurrentBuildInfo() BuildInfo {
	return BuildInfo{Version: Version, Commit: Commit, BuildDate: BuildDate}
}
