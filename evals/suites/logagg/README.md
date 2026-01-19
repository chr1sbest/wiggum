# Log Aggregator Eval Suite

Tests for evaluating Log Aggregator CLI implementations.

## Test Approach

1. **Build test**: Does `go build` succeed?
2. **Parse test**: Can it parse JSON/Apache/Syslog logs?
3. **Filter test**: Does filtering by level/time work?
4. **Stats test**: Are aggregations correct?
5. **Query test**: Does SQL-like query work?

## Fixtures

- `fixtures/json.log` - JSON format logs
- `fixtures/apache.log` - Apache Common format
- `fixtures/syslog.log` - Syslog format

## Running

```bash
./run_tests.sh <project_dir>
```
