package config

type Config struct {
	ListenPort          int    `mapstructure:"listen-port"`
	MainTarget          string `mapstructure:"main-target"`
	CanaryTarget        string `mapstructure:"canary-target"`
	SidecarUrl          string `mapstructure:"sidecar-url"`
	MainSidecarStatus   int    `mapstructure:"main-sidecar-status"`
	CanarySidecarStatus int    `mapstructure:"canary-sidecar-status"`
}
