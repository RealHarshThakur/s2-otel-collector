package s2otelexporter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/s2-streamstore/s2-sdk-go/s2"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// maxRecordSizeBytes is the maximum size for a single record.
// S2 spec: each record may be up to 1 MiB in metered bytes.
const maxRecordSizeBytes = 1 * 1024 * 1024 // 1 MiB max per record

type batchKey struct {
	service string
	ns      string
	hour    string
}

type basinExporter struct {
	logger             *zap.Logger
	basin              string
	streamPrefix       string
	accessToken        string
	basinClient        *s2.BasinClient
	resourceAttributes ResourceAttributesConfig
	batch              BatchConfig
}

func (b *basinExporter) start(ctx context.Context, host component.Host) error {
	b.logger.Info("Starting Basin exporter", zap.String("basin", b.basin))

	b.accessToken = os.Getenv("S2_ACCESS_TOKEN")

	basinClient, err := s2.NewBasinClient(b.basin, b.accessToken)
	if err != nil {
		return err
	}

	b.basinClient = basinClient
	return nil
}

func (b *basinExporter) shutdown(ctx context.Context) error {
	b.logger.Info("Shutting down Basin exporter")
	return nil
}

// serializeLogRecord converts an OpenTelemetry log record to JSON with all core fields.
func serializeLogRecord(record plog.LogRecord, resource pcommon.Resource) []byte {
	recordAttrs := make(map[string]interface{})
	record.Attributes().Range(func(k string, v pcommon.Value) bool {
		recordAttrs[k] = v.AsString()
		return true
	})

	resourceAttrs := make(map[string]interface{})
	resource.Attributes().Range(func(k string, v pcommon.Value) bool {
		resourceAttrs[k] = v.AsString()
		return true
	})

	// Build the structured log object
	logEntry := map[string]interface{}{
		"timestamp":     record.Timestamp().AsTime().Format(time.RFC3339Nano),
		"severity_text": record.SeverityText(),
		"body":          record.Body().AsString(),
		"attributes":    recordAttrs,
		"resource":      resourceAttrs,
	}

	data, err := json.Marshal(logEntry)
	if err != nil {
		// Fallback to just the body if marshaling fails
		return []byte(record.Body().AsString())
	}
	return data
}

func (b *basinExporter) pushLogs(ctx context.Context, ld plog.Logs) error {
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		resource := rls.At(i).Resource()

		var svcName, nsName string
		if b.resourceAttributes.ServiceNameAttribute != "" {
			if val, ok := resource.Attributes().Get(b.resourceAttributes.ServiceNameAttribute); ok {
				svcName = val.AsString()
			}
		}
		if b.resourceAttributes.NamespaceAttribute != "" {
			if val, ok := resource.Attributes().Get(b.resourceAttributes.NamespaceAttribute); ok {
				nsName = val.AsString()
			}
		}

		sl := rls.At(i).ScopeLogs()
		for j := 0; j < sl.Len(); j++ {
			logRecords := sl.At(j).LogRecords()
			for k := 0; k < logRecords.Len(); k++ {
				record := logRecords.At(k)

				serializedBody := serializeLogRecord(record, resource)

				if len(serializedBody) > maxRecordSizeBytes {
					b.logger.Warn("Discarding log record: exceeds max record size",
						zap.String("service", svcName),
						zap.String("namespace", nsName),
						zap.Int("record_size_bytes", len(serializedBody)),
						zap.Int("max_record_size_bytes", maxRecordSizeBytes),
					)
					continue
				}

				timestamp := record.Timestamp().AsTime()
				if timestamp.IsZero() {
					timestamp = time.Now()
				}
				hourKey := timestamp.UTC().Format("2006-01-02-15")
				streamName := fmt.Sprintf("%s-%s-%s", b.streamPrefix, svcName, hourKey)

				if err := b.sendRecord(ctx, streamName, serializedBody); err != nil {
					b.logger.Error("Failed to send log record to stream",
						zap.String("stream", streamName),
						zap.Error(err),
					)
					return err
				}
			}
		}
	}
	return nil
}

func (b *basinExporter) sendRecord(ctx context.Context, streamName string, body []byte) error {
	streamClient, err := s2.NewStreamClient(b.basin, streamName, b.accessToken)
	if err != nil {
		return fmt.Errorf("failed to create stream client: %w", err)
	}

	batch, _ := s2.NewAppendRecordBatch(s2.AppendRecord{Body: body})

	_, err = streamClient.Append(ctx, &s2.AppendInput{Records: batch})
	if err != nil {
		return fmt.Errorf("failed to append record: %w", err)
	}

	b.logger.Debug("Sent log record to stream",
		zap.String("stream", streamName),
		zap.Int("body_size_bytes", len(body)),
	)
	return nil
}
