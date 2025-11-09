// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package s2otelexporter // import "github.com/realharshthakur/s2-otel-collector/s2-otel-exporter"

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// TypeStr is the unique identifier for your exporter type.
const TypeStr = "basin"

// CreateDefaultConfig returns the default configuration for the exporter.
func CreateDefaultConfig() component.Config {
	return &Config{
		StreamPrefix: "",
		ResourceAttributes: ResourceAttributesConfig{
			ServiceNameAttribute: "service.name",
			NamespaceAttribute:   "k8s.namespace.name",
		},
		Batch: BatchConfig{
			MaxRecords:    1000,
			FlushInterval: 2 * time.Second,
		},
	}
}

// CreateLogsExporter creates a new Basin Logs exporter instance.
func CreateLogsExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	if _, ok := os.LookupEnv("S2_ACCESS_TOKEN"); !ok {
		return nil, fmt.Errorf("S2_ACCESS_TOKEN environment variable is not set")
	}

	c := cfg.(*Config)

	exp := &basinExporter{
		logger:             set.Logger,
		basin:              c.BasinName,
		streamPrefix:       c.StreamPrefix,
		resourceAttributes: c.ResourceAttributes,
		batch:              c.Batch,
	}

	//// exporterhelper settings chosen to respect these constraints:
	//   • queue.num_consumers = 4      → prevents >200 appends/sec per stream
	//   • queue.queue_size = 2000      → buffers transient spikes safely
	//   • retry.backoff (200 ms–5 s)   → avoids hammering after 429 throttling
	//   • timeout = 10 s               → allows S2 network round-trip before abort

	queueCfg := exporterhelper.NewDefaultQueueConfig()
	queueCfg.NumConsumers = 4
	queueCfg.QueueSize = 2000

	retryCfg := configretry.NewDefaultBackOffConfig()
	retryCfg.InitialInterval = 200 * time.Millisecond
	retryCfg.MaxInterval = 5 * time.Second
	retryCfg.MaxElapsedTime = 30 * time.Second

	return exporterhelper.NewLogs(
		ctx,
		set,
		cfg,
		exp.pushLogs,
		exporterhelper.WithStart(exp.start),
		exporterhelper.WithShutdown(exp.shutdown),
		exporterhelper.WithRetry(retryCfg),
		exporterhelper.WithQueue(queueCfg),
		exporterhelper.WithTimeout(exporterhelper.NewDefaultTimeoutConfig()),
	)
}

// NewFactory registers the Basin exporter as an OTEL component.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		component.MustNewType(TypeStr),
		CreateDefaultConfig,
		exporter.WithLogs(CreateLogsExporter, component.StabilityLevelAlpha),
	)
}
