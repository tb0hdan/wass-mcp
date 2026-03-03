package types

const (
	// DefaultHost is the default target host for scanner tools.
	DefaultHost = "localhost"
	// DefaultPort is the default target port for scanner tools.
	DefaultPort = 80
	// HTTPSPort is the standard HTTPS port.
	HTTPSPort = 443

	// SchemeHTTP is the HTTP URL scheme.
	SchemeHTTP = "http"
	// SchemeHTTPS is the HTTPS URL scheme.
	SchemeHTTPS = "https"

	// MaxDefaultLines is the default maximum lines to return for top view.
	MaxDefaultLines = 200
	// MaxAllowedLines is the maximum allowed lines limit for pagination.
	MaxAllowedLines = 100000
)
