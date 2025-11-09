package s2otelexporter

import (
	"fmt"
	"time"
)

type Config struct {
	// BasinName is the name of the Basin to which logs will be sent.
	BasinName string `mapstructure:"basin_name"`

	// StreamPrefix is an optional prefix for stream names.
	StreamPrefix string `mapstructure:"stream_prefix"`

	// ResourceAttributes defines which resource attributes to use for stream naming.
	ResourceAttributes ResourceAttributesConfig `mapstructure:"resource_attributes"`

	// Batch controls batching behavior before sending logs to the Basin.
	Batch BatchConfig `mapstructure:"batch"`
}

// ResourceAttributesConfig defines which resource attributes to include in stream names.
type ResourceAttributesConfig struct {
	// ServiceNameAttribute is the resource attribute key for service name. Set to empty to disable.
	// Default: "service.name"
	ServiceNameAttribute string `mapstructure:"service_name_attribute"`

	// NamespaceAttribute is the resource attribute key for namespace. Set to empty to disable.
	// Default: "k8s.namespace.name"
	NamespaceAttribute string `mapstructure:"namespace_attribute"`
}

// BatchConfig defines parameters for batching logs before flushing to the backend.
type BatchConfig struct {
	// MaxRecords defines how many log records can be grouped before a flush.
	// Default: 1000
	MaxRecords int `mapstructure:"max_records"`

	// FlushInterval defines how long to wait before flushing pending logs, even if MaxRecords isn’t reached.
	// Default: 2s
	FlushInterval time.Duration `mapstructure:"flush_interval"`
}

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	if cfg.BasinName == "" {
		return fmt.Errorf("basin_name cannot be empty")
	}
	return nil
}
