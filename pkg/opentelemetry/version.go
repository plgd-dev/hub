package opentelemetry

const InstrumentationName = "github.com/plgd-dev/hub/pkg/opentelemetry"

// Version is the current release version of the plgd instrumentation.
func Version() string {
	return "0.0.1"
}

// SemVersion is the semantic version to be supplied to tracer/meter creation.
func SemVersion() string {
	return "semver:" + Version()
}
