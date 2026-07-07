# CSV to JSON Utility

A high-performance, streaming, RFC 4180-compliant CSV to JSON converter written in Go.

## Features

- **RFC 4180 Compliance**: Supports complex, standard CSV features like escaped double-quotes and multi-line fields.
- **Header Mapping**: Interprets the first row of input as the keys for the generated JSON objects.
- **Custom Delimiters**: Supports arbitrary custom delimiters (e.g. `;`, `\t`, `|`, etc.) via command-line flags.
- **Data Type Inference**: Auto-detects integers, floats, booleans, and null values, preventing numeric/boolean values from being encoded as strings.
- **Empty Value Handling**: Flexible handling of empty cells through three distinct strategies:
  - `null`: Maps empty cells to JSON `null` (default).
  - `omit`: Removes the field from the resulting JSON object.
  - `empty`: Retains the field with an empty string (`""`).
- **Memory-Efficient I/O Streaming**: Implements row-by-row streaming to maintain $O(1)$ memory complexity, capable of processing datasets larger than 1GB.
- **Multiple Encoding Support**: Supports UTF-8, ISO-8859-1, and UTF-16 (BE/LE with BOM auto-detection).
- **Error Handing**: Logs warning and skips malformed rows by default, or aborts immediately on mismatched column count/syntax errors via flag.

## Installation

Since the package is part of the `harness-app` repository, you can import it in Go:

```go
import "github.com/dothanhlam/harness-app/csv_to_json"
```

## Example Usage

### Library API

```go
package main

import (
	"os"
	"strings"
	
	"github.com/dothanhlam/harness-app/csv_to_json"
)

func main() {
	input := "name,age,active\nAlice,30,true\n"
	r := strings.NewReader(input)
	w := os.Stdout
	
	opts := csv_to_json.Options{
		Delimiter:     ',',
		InferTypes:    true,
		EmptyStrategy: csv_to_json.StrategyNull,
		Encoding:      "utf-8",
		Pretty:        true,
	}
	
	_ = csv_to_json.Convert(r, w, opts)
}
```

### CLI Execution

```go
package main

import (
	"os"

	"github.com/dothanhlam/harness-app/csv_to_json"
)

func main() {
	// Execute CLI with flags
	os.Exit(csv_to_json.RunCLI(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
```

#### CLI Command Flags
- `-i, --input <path>`: Source CSV file path (reads from `stdin` if omitted).
- `-o, --output <path>`: Destination JSON file path (writes to `stdout` if omitted).
- `-d, --delimiter <char>`: Custom delimiter (default: `,`).
- `-e, --empty-strategy <omit|null|empty>`: Handle empty cells (default: `null`).
- `-n, --infer-types <true|false>`: Enable type inference (default: `true`).
- `-c, --encoding <utf-8|iso-8859-1|utf-16>`: Source encoding (default: `utf-8`).
- `-a, --abort-on-error`: Abort processing if a row has mismatched columns or syntax error (default: `false`).
- `--pretty`: Format/pretty-print JSON output.

### Exit Codes
- `0`: Success.
- `1`: File Not Found or Permission Denied.
- `2`: Malformed CSV Syntax / Invalid Arguments.
- `3`: Disk I/O or Space Error.

## Example Input vs. Output

### Input CSV (`data.csv`)
```csv
id,name,rating,active,notes
1,Alice,4.9,true,
2,Bob,4.2,false,some description
3,Charlie,3.8,,
```

### Output JSON (using `--pretty` and `--empty-strategy null`)
```json
[
  {
    "id": 1,
    "name": "Alice",
    "rating": 4.9,
    "active": true,
    "notes": null
  },
  {
    "id": 2,
    "name": "Bob",
    "rating": 4.2,
    "active": false,
    "notes": "some description"
  },
  {
    "id": 3,
    "name": "Charlie",
    "rating": 3.8,
    "active": null,
    "notes": null
  }
]
```

## Performance & Scalability

### Benchmarks
- Time Complexity: $O(N)$ where $N$ is the size of the input stream.
- Space Complexity: $O(1)$ constant memory utilization (below 10MB RSS for files > 1GB) due to streaming chunked parsing.
