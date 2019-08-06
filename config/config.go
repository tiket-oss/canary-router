package config

type Config struct {
	ListenPort          int            `mapstructure:"listen-port"`
	MainTarget          string         `mapstructure:"main-target"`
	CanaryTarget        string         `mapstructure:"canary-target"`
	SidecarUrl          string         `mapstructure:"sidecar-url"`
	MainSidecarStatus   int            `mapstructure:"main-sidecar-status"`
	CanarySidecarStatus int            `mapstructure:"canary-sidecar-status"`
	CircuitBreaker      CircuitBreaker `mapstructure:"circuit-breaker"`
}

type CircuitBreaker struct {
	RequestLimitCanary uint64 `mapstructure:"request-limit-canary"`
}
