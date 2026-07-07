# JSON Formatter Package

A premium, high-performance, and responsive JSON formatter, minifier, and structural validator. Implemented as an isolated Go package that embeds a gorgeous, modern web interface.

## Features

- **Valid JSON Parsing**: Complete support for standard ECMA-404 JSON parsing.
- **Modern Web UI**: Features a beautiful glassmorphic dark interface with neon night glow, multiple user-selectable themes, and responsive design layouts.
- **Custom Indentation Options**: Format output with 2 spaces, 4 spaces, or Tab characters.
- **Compact Minifier**: Minify large JSON objects down to single-line formats.
- **Drag & Drop and File Picker**: Easily import external `.json` and `.txt` files directly into the raw editor pane.
- **Structure Metrics Analyzer**: Real-time stats calculation including File Size, Maximum Nesting Depth, Total Keys, and Object Count.
- **Detailed Location-Based Syntax Errors**: In case of invalid JSON syntax, precisely locate the error down to the exact Line, Column, and Offset.
- **Copy and Download Actions**: Quick clipboard copying with toast confirmation, and JSON output downloads.

## Technical Structure

```
workspace/json_formatter/
├── README.md
├── json_formatter.go
├── json_formatter_test.go
└── static/
    ├── index.html
    ├── style.css
    └── app.js
```

## Go API Usage

### Handler

The package exposes a `Handler()` function which returns a self-contained `http.Handler` routing requests to both the embedded static web assets and the JSON processing API.

```go
import (
    "net/http"
    "github.com/dothanhlam/harness-app/json_formatter"
)

func main() {
    handler := json_formatter.Handler()
    http.ListenAndServe(":8080", handler)
}
```

### JSON API: POST `/api/format`

Allows formatting payloads programmatically:

#### Request Body
```json
{
  "jsonData": "{\"name\":\"Antigravity\"}",
  "indent": 2,
  "minify": false
}
```

#### Successful Response
```json
{
  "success": true,
  "formatted": "{\n  \"name\": \"Antigravity\"\n}",
  "stats": {
    "size": 22,
    "maxDepth": 1,
    "keyCount": 1,
    "arrayCount": 0,
    "objectCount": 1
  }
}
```

#### Syntax Error Response
```json
{
  "success": false,
  "errorMessage": "invalid character '}' looking for beginning of object key string",
  "errorOffset": 12,
  "errorLine": 1,
  "errorCol": 12
}
```

## Unit Testing

Run standard tests via:

```bash
go test -v ./workspace/json_formatter
```
