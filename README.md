# S2 OpenTelemetry Collector

This is an OpenTelemetry collector with a custom S2 Basin exporter that sends log data to S2 StreamStore.

## Features

- **OTLP Receiver**: Accepts OpenTelemetry Protocol (OTLP) logs via gRPC and HTTP
- **S2 Exporter**: Sends logs to S2 Basin in JSON format with full structured data
- **Batch Processor**: Batches logs before sending for efficient delivery
- **Debug Exporter**: Optional debug output for inspection

## Quick Start

1. Get an access token from [S2 Dashboard](https://s2.dev/dashboard)
2. Create a basin: `s2 create-basin my-logs`
3. Configure the collector in `config.yaml` with your basin name
4. Set your token: `export S2_ACCESS_TOKEN=your_token`
5. Run: `go run ./otelcol-dev --config config.yaml`
6. Send logs via OTLP to `http://localhost:4318/v1/logs`

For detailed setup instructions, see the [Prerequisites](#prerequisites) section below.

## Prerequisites

### 1. S2 Account & Access Token

Follow the [S2 Quickstart](https://s2.dev/docs/quickstart) to:

1. **Create an account** and log in to [S2 Dashboard](https://s2.dev/dashboard)
2. **Create an organization**
3. **Issue an access token** in the dashboard settings
4. **Set the token** in your environment:
   ```bash
   export S2_ACCESS_TOKEN=your_token_here
   ```

### 2. Create a Basin

Basins are globally unique storage containers. Using the [S2 CLI](https://s2.dev/docs/cli):

```bash
# Choose a unique basin name (8-48 chars, lowercase letters, numbers, hyphens)
export BASIN="my-logs-basin"
s2 create-basin ${BASIN}
```

For more information, see the [S2 Basin documentation](https://s2.dev/docs/basin).

### 3. Runtime Requirements

- **Go 1.24+**: Required to build and run the collector

## Configuration

Edit `config.yaml` to configure:

- **Basin Name**: The S2 basin to send logs to (e.g., `my-basin`)
- **Stream Prefix**: Prefix for stream names (e.g., `dev` will create streams like `dev-service-2024-11-09-07-0`)
- **Resource Attributes**: Which resource attributes to use in stream names
- **Batch Settings**: Log batching parameters

### Stream Naming

Streams are named based on:
```
{stream_prefix}-{service.name}-{namespace}-{hour}
```

For example, with stream prefix `dev` and service `frontend`:
```
dev-frontend--2024-11-09-07
```


## Running the Collector

```bash
go run ./otelcol-dev --config config.yaml
```

The collector will start listening on:
- **gRPC**: `127.0.0.1:4317`
- **HTTP**: `127.0.0.1:4318`

## Testing

### 1. Start the Collector

In one terminal:
```bash
go run ./otelcol-dev --config config.yaml
```

### 2. Send Test Logs

In another terminal, use curl to send OTLP logs:

```bash
curl -X POST http://localhost:4318/v1/logs \
  -H "Content-Type: application/json" \
  -d '{
    "resourceLogs": [{
      "resource": {
        "attributes": [
          { "key": "service.name", "value": { "stringValue": "frontend" } }
        ]
      },
      "scopeLogs": [{
        "scope": { "name": "test-scope" },
        "logRecords": [
          {
            "body": { "stringValue": "Hello Basin!" },
            "timeUnixNano": "'$(date +%s)'000000000",
            "severityText": "INFO"
          }
        ]
      }]
    }]
  }'
```

### 3. Read Logs from S2

In another terminal, read the logs back using the S2 CLI:

```bash
s2 read s2://{basin-name}/{stream-name}
```

Example (adjust based on your config):
```bash
s2 read s2://my-basin/dev-frontend--2024-11-09-07
```

To get the exact stream name, check the collector logs when data is sent. The debug exporter will also show the data being processed.

## Log Format

Each log entry in S2 is stored as JSON with the following structure:

```json
{
  "timestamp": "2024-11-09T07:30:20.123456789+05:30",
  "severity_text": "INFO",
  "body": "Hello Basin!",
  "attributes": {
    "custom_key": "custom_value"
  },
  "resource": {
    "service.name": "frontend",
    "service.instance.id": "instance-123"
  }
}
```

## Troubleshooting


### "S2_ACCESS_TOKEN environment variable is not set"
- Set the token: `export S2_ACCESS_TOKEN=your_token_here`

### Basin not found
- Verify the `basin_name` in `config.yaml` matches an existing basin
- Check S2 access permissions

### No streams created
- Check collector logs for errors
- Ensure logs are being sent to the correct endpoint (HTTP: `:4318`, gRPC: `:4317`)
