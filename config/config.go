package config

// Config holds the configuration values to be used throughout the application.
type Config struct {
	MainTarget   string `mapstructure:"main-target"`
	CanaryTarget string `mapstructure:"canary-target"`
	SidecarURL   string `mapstructure:"sidecar-url"`

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
}

// InstrumentationConfig holds the configuration values specific to the instrumentation aspect.
type InstrumentationConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

// CircuitBreaker holds the configuration values specific to the circuit breaking aspect.
type CircuitBreaker struct {
	RequestLimitCanary uint64 `mapstructure:"request-limit-canary"`
}

// HTTPServerConfig holds the configuration for instantiating http.Server
type HTTPServerConfig struct {
	Host         string `mapstructure:"host"`
	Port         string `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read-timeout"`
	WriteTimeout int    `mapstructure:"write-timeout"`
	IdleTimeout  int    `mapstructure:"idle-timeout"`
}

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
}
