package version

var (
	Version string
)

func init() {
	if Version == "" {
		Version = "unknown"
	}
}
