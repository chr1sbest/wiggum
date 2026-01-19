# Log Aggregator CLI (Advanced)

Build a production-ready CLI tool for parsing, filtering, aggregating, and alerting on log files. Supports multiple log formats, real-time watching, and webhook notifications.

## Technical Stack
- Go 1.21+
- Cobra for CLI framework
- Viper for configuration
- Standard library for file I/O and JSON parsing
- testify for testing assertions
- No external database (in-memory processing)

## Core Features

### Log Format Parsing
Support parsing these log formats:
- **JSON**: One JSON object per line with arbitrary fields
- **Apache Common**: `127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326`
- **Apache Combined**: Common + referer and user-agent
- **Syslog (RFC 3164)**: `<34>Oct 11 22:14:15 mymachine su: 'su root' failed for lonvick on /dev/pts/8`
- **Custom regex**: User-defined pattern with named capture groups

### Field Extraction
For all formats, extract and normalize:
- `timestamp` (parse to time.Time, support multiple date formats)
- `level` (info, warn, error, debug, fatal - normalize case)
- `message` (the log message body)
- `source` (hostname, service name, or filename)
- Additional fields vary by format (e.g., `status_code`, `method`, `path` for Apache)

## CLI Commands

### `logagg parse <file>`
Parse and display logs in normalized format.
- `--format <json|apache|combined|syslog|custom>` (default: auto-detect)
- `--pattern <regex>` (required if format=custom)
- `--output <json|table|csv>` (default: table)
- `--fields <field1,field2,...>` (which fields to display)

### `logagg filter <file>`
Filter logs by various criteria.
- `--level <level>` (filter by log level, supports comma-separated list)
- `--from <timestamp>` (logs after this time, supports relative like "1h ago")
- `--to <timestamp>` (logs before this time)
- `--match <regex>` (message matches regex)
- `--field <name>=<value>` (exact field match, can repeat)
- `--field <name>~<regex>` (field matches regex)
- `--invert` (invert match, show non-matching lines)
- Output options same as parse

### `logagg stats <file>`
Generate aggregate statistics.
- `--group-by <field>` (group stats by field, e.g., level, source, status_code)
- `--interval <duration>` (time bucket size: 1m, 5m, 1h, 1d)
- Output includes:
  - Total log count
  - Count by level
  - Error rate (errors / total)
  - Logs per second/minute
  - Top N values for specified fields
  - Latency percentiles if `duration` or `response_time` field exists (p50, p90, p99)

### `logagg watch <file>`
Tail a log file with live filtering and alerting.
- All filter options from `logagg filter`
- `--config <file>` (YAML config with alert rules)
- `--webhook <url>` (send alerts to webhook)
- `--follow` / `-f` (continue watching for new lines, default true)
- `--lines <n>` / `-n` (show last N lines before watching)
- Handle log rotation (detect file truncation/rename, reopen)

### `logagg query <file> <query>`
SQL-like query language for logs.
- Examples:
  - `SELECT level, count(*) FROM logs GROUP BY level`
  - `SELECT * FROM logs WHERE level = 'error' AND timestamp > '2024-01-01'`
  - `SELECT source, avg(response_time) FROM logs GROUP BY source ORDER BY avg(response_time) DESC LIMIT 10`
- Support: SELECT, FROM, WHERE, GROUP BY, ORDER BY, LIMIT
- Aggregate functions: count, sum, avg, min, max, p50, p90, p99

### `logagg merge <file1> <file2> ...`
Merge multiple log files.
- `--output <file>` (output file, default stdout)
- `--sort` (sort by timestamp, default true)
- `--dedup` (remove duplicate lines based on timestamp + message hash)
- Handle files with different formats

## Configuration

### Config File (`~/.logagg.yaml` or `--config`)
```yaml
defaults:
  format: auto
  output: table
  timezone: UTC

formats:
  myapp:
    pattern: '^\[(?P<timestamp>[^\]]+)\] (?P<level>\w+) (?P<message>.*)$'
    timestamp_format: "2006-01-02 15:04:05.000"

alerts:
  - name: high_error_rate
    condition: "error_rate > 0.05"
    window: 5m
    cooldown: 10m
    
  - name: slow_responses
    condition: "p99(response_time) > 1000"
    window: 1m
    cooldown: 5m
    
  - name: error_spike
    condition: "count(level='error') > 100"
    window: 1m
    cooldown: 15m

webhooks:
  slack:
    url: "${SLACK_WEBHOOK_URL}"
    template: |
      {"text": "🚨 Alert: {{.Name}} triggered\n{{.Message}}"}
```

## Alert System

### Alert Conditions
- Rate-based: `error_rate > 0.1` (errors per total in window)
- Count-based: `count(level='error') > 50`
- Percentile-based: `p99(response_time) > 2000`
- Threshold: `avg(response_time) > 500`

### Alert Actions
- Print to stderr with timestamp and details
- Send to configured webhook with JSON payload
- Write to alert log file

### Alert Payload (Webhook)
```json
{
  "name": "high_error_rate",
  "triggered_at": "2024-01-15T10:30:00Z",
  "condition": "error_rate > 0.05",
  "actual_value": 0.08,
  "window": "5m",
  "sample_logs": ["error log 1...", "error log 2..."]
}
```

### Cooldown
- Don't re-trigger same alert within cooldown period
- Reset cooldown if condition returns to normal, then triggers again

## Error Handling
- Invalid log lines: skip and count (report at end)
- Missing files: error with clear message
- Invalid regex: error at startup with position indicator
- Parse errors: include line number and raw content
- Network errors (webhooks): retry 3 times with exponential backoff

## Output Formats

### Table (default)
```
TIMESTAMP            LEVEL  SOURCE    MESSAGE
2024-01-15 10:30:00  ERROR  api       Connection refused to database
2024-01-15 10:30:01  INFO   api       Retrying connection...
```

### JSON
```json
{"timestamp": "2024-01-15T10:30:00Z", "level": "error", "source": "api", "message": "..."}
```

### CSV
```
timestamp,level,source,message
2024-01-15T10:30:00Z,error,api,"Connection refused to database"
```

## Performance Requirements
- Handle files up to 1GB without loading entirely into memory
- Stream processing for watch mode
- Parse at least 100k lines/second on typical hardware
- Efficient regex compilation (compile once, reuse)

## Tests Required

### Parser Tests
- JSON log parsing (valid, malformed, nested objects)
- Apache Common format (valid, edge cases with quotes and special chars)
- Apache Combined format
- Syslog format (various priority levels, timestamps)
- Custom regex patterns (capture groups, optional fields)
- Timestamp parsing (multiple formats, timezones)
- Auto-detection of log formats

### Filter Tests
- Level filtering (single, multiple, case insensitive)
- Time range filtering (absolute, relative timestamps)
- Regex matching (simple, complex patterns, edge cases)
- Field matching (exact, regex)
- Filter combination (AND logic)
- Inverted matching

### Stats Tests
- Count aggregation by field
- Error rate calculation
- Time bucketing (1m, 5m, 1h intervals)
- Percentile calculations (p50, p90, p99)
- Empty input handling
- Large dataset handling (streaming)

### Query Tests
- SELECT with field list
- WHERE with various operators
- GROUP BY single and multiple fields
- ORDER BY with ASC/DESC
- LIMIT
- Aggregate functions
- Query parsing errors

### Watch Tests
- File tailing (new lines appended)
- Log rotation handling
- Alert triggering
- Webhook delivery (mock HTTP server)
- Cooldown behavior

### Merge Tests
- Two files with interleaved timestamps
- Multiple files
- Deduplication
- Different formats in same merge

### Integration Tests
- End-to-end: generate logs → parse → filter → stats
- Config file loading
- Environment variable expansion in config
- CLI flag precedence over config

## Build & Run

### Build
```bash
go build -o logagg ./cmd/logagg
```

### Example Usage
```bash
# Parse and display
./logagg parse app.log --format json --output table

# Filter errors from last hour
./logagg filter app.log --level error --from "1h ago"

# Get stats grouped by level
./logagg stats app.log --group-by level --interval 5m

# Watch with alerts
./logagg watch app.log --config alerts.yaml --webhook https://hooks.slack.com/...

# SQL-like query
./logagg query app.log "SELECT level, count(*) FROM logs GROUP BY level"

# Merge and dedupe
./logagg merge app1.log app2.log --dedup --output merged.log
```

## Project Structure
```
cmd/
  logagg/
    main.go
internal/
  parser/
    parser.go
    json.go
    apache.go
    syslog.go
    custom.go
  filter/
    filter.go
  stats/
    stats.go
    percentile.go
  query/
    lexer.go
    parser.go
    executor.go
  alert/
    alert.go
    webhook.go
  config/
    config.go
  output/
    table.go
    json.go
    csv.go
go.mod
go.sum
README.md
```

## Deliverables
- All source code with proper package structure
- go.mod with dependencies
- README.md with usage examples
- Example config file
- Comprehensive test suite (go test ./...)
- Sample log files for testing
