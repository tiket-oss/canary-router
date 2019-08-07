package config

// Config holds the configuration values to be used throughout the application.
type Config struct {
	ListenPort   int    `mapstructure:"listen-port"`
	MainTarget   string `mapstructure:"main-target"`
	CanaryTarget string `mapstructure:"canary-target"`
	SidecarURL   string `mapstructure:"sidecar-url"`

	// MainSidecarStatus is the expected HTTP Status code that will be returned by Sidecar
	// should the route be passed to Main service.
	MainSidecarStatus int `mapstructure:"main-sidecar-status"`

	// CanarySidecarStatus is the expected HTTP Status code that will be returned by Sidecar
	// should the route be passed to Canary service.
	CanarySidecarStatus int `mapstructure:"canary-sidecar-status"`

	CircuitBreaker  CircuitBreaker        `mapstructure:"circuit-breaker"`
	Instrumentation InstrumentationConfig `mapstructure:"instrumentation"`
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
