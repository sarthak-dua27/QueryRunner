# QueryRunner

QueryRunner is a Go-based tool designed to execute a batch of queries against a Couchbase Full-Text Search (FTS) endpoint, leveraging concurrency for efficient query execution. The results are stored in a JSON file for further analysis.

## Usage

Run the program with the following command:

```bash
go run . -host http://<ip>:8094 -user <username> -pass <password> -index <indexName> -concurrency <numGoroutines> -iterations <numIterations> -numqueries <totalQueries> -print-results <true/false>
```

## Parameters
- **`-host`**: The Couchbase FTS endpoint (e.g., `http://127.0.0.1:8094`).
- **`-user`**: Couchbase usernamee.
- **`-pass`**: Couchbase password.
- **`-index`**: Name of the FTS index to query.
- **`-concurrency`**: Number of concurrent goroutines to use for query execution.
- **`-iterations`**: Number of times to repeat each query.
- **`-numqueries`**: Total number of queries to execute. Must be a multiple of 3.
- **`-print-results`**: Set to `true` to write query results to `results.json`.

## Example Output

```code
Successful: 300
Failed: 0
Results written to results.json
```