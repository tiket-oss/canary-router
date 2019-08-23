package config

// Config holds the configuration values to be used throughout the application.
// TODO: add logrus level
type Config struct {
	MainTarget       string `mapstructure:"main-target"`
	MainHeaderHost   string `mapstructure:"main-header-host"`
	CanaryTarget     string `mapstructure:"canary-target"`
	CanaryHeaderHost string `mapstructure:"canary-header-host"`
	SidecarURL       string `mapstructure:"sidecar-url"`

	// TrimPrefix if set will modify subsequent request path to main, canary, and sidecar service
	// by removing TrimPrefix substring in the request path string
	TrimPrefix string `mapstructure:"trim-prefix"`

	// MainSidecarStatus is the expected HTTP Status code that will be returned by Sidecar
	// should the route be passed to Main service.
	MainSidecarStatus int `mapstructure:"main-sidecar-status"`

	// CanarySidecarStatus is the expected HTTP Status code that will be returned by Sidecar
	// should the route be passed to Canary service.
	CanarySidecarStatus int `mapstructure:"canary-sidecar-status"`

	CircuitBreaker  CircuitBreaker        `mapstructure:"circuit-breaker"`
	Instrumentation InstrumentationConfig `mapstructure:"instrumentation"`
	Server          HTTPServerConfig      `mapstructure:"router-server"`
	Client          MultiHTTPClientConfig `mapstructure:"proxy-client"`

	Log Log `mapstructure:"log"`
}

// InstrumentationConfig holds the configuration values specific to the instrumentation aspect.
type InstrumentationConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

// CircuitBreaker holds the configuration values specific to the circuit breaking aspect.
type CircuitBreaker struct {
	RequestLimitCanary uint64 `mapstructure:"request-limit-canary"`
	ErrorLimitCanary   uint64 `mapstructure:"error-limit-canary"`
}

// HTTPServerConfig holds the configuration for instantiating http.Server
type HTTPServerConfig struct {
	Host         string `mapstructure:"host"`
	Port         string `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read-timeout"`
	WriteTimeout int    `mapstructure:"write-timeout"`
	IdleTimeout  int    `mapstructure:"idle-timeout"`
}

// MultiHTTPClientConfig holds the configuration for instantiating main&canary and sidecar proxy http.Client
type MultiHTTPClientConfig struct {
	MainAndCanary HTTPClientConfig `mapstructure:"to-main-and-canary"`
	Sidecar       HTTPClientConfig `mapstructure:"to-sidecar"`
}

// HTTPClientConfig holds the configuration for instantiating http.Client
type HTTPClientConfig struct {
	Timeout            int  `mapstructure:"timeout"`
	MaxIdleConns       int  `mapstructure:"max-idle-conns"`
	IdleConnTimeout    int  `mapstructure:"idle-conn-timeout"`
	DisableCompression bool `mapstructure:"disable-compression"`
	TLS                TLS  `mapstructure:"tls"`
}

// TLS holds the configuration of TLS
type TLS struct {
	InsecureSkipVerify bool `mapstructure:"insecure-skip-verify"`
}

// Log holds the configuration values specific to the logging aspect.
type Log struct {
	Level            string `mapstructure:"level"`
	DebugRequestBody bool   `mapstructure:"debug-request-body"`
}
