// Package csv_to_json provides a streaming, RFC 4180-compliant CSV to JSON converter.
// It supports custom delimiters, primitive data type inference, empty value handling strategies,
// and multiple file encodings (UTF-8, ISO-8859-1, and UTF-16).
package csv_to_json

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// EmptyStrategy represents the handling strategy for empty CSV cells.
type EmptyStrategy string

const (
	// StrategyOmit removes the key from the output JSON object if the cell is empty.
	StrategyOmit EmptyStrategy = "omit"
	// StrategyNull sets the key value to JSON null if the cell is empty.
	StrategyNull EmptyStrategy = "null"
	// StrategyEmpty sets the key value to an empty string if the cell is empty.
	StrategyEmpty EmptyStrategy = "empty"
)

// Options configuration for the conversion process.
type Options struct {
	Delimiter     rune
	InferTypes    bool
	EmptyStrategy EmptyStrategy
	AbortOnError  bool
	Pretty        bool
	Encoding      string // "utf-8", "iso-8859-1", "utf-16"
}

// isoReader converts ISO-8859-1 stream into UTF-8 stream in a memory-efficient way.
type isoReader struct {
	r   io.Reader
	buf []byte
}

func (ir *isoReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(ir.buf) == 0 {
		var temp [4096]byte
		n, err := ir.r.Read(temp[:])
		if n == 0 {
			return 0, err
		}
		var utf8Buf []byte
		for i := 0; i < n; i++ {
			r := rune(temp[i])
			var rBuf [4]byte
			wn := utf8.EncodeRune(rBuf[:], r)
			utf8Buf = append(utf8Buf, rBuf[:wn]...)
		}
		ir.buf = utf8Buf
	}
	n := copy(p, ir.buf)
	ir.buf = ir.buf[n:]
	return n, nil
}

// utf16Reader converts UTF-16 (BE/LE) with BOM stream into UTF-8 stream in a memory-efficient way.
type utf16Reader struct {
	r        io.Reader
	isLE     bool
	buf      []byte
	readBOM  bool
	leftover []byte
}

func (ur *utf16Reader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(ur.buf) == 0 {
		var temp [4096]byte
		start := 0
		if len(ur.leftover) > 0 {
			temp[0] = ur.leftover[0]
			start = 1
			ur.leftover = nil
		}
		n, err := ur.r.Read(temp[start:])
		total := start + n
		if total == 0 {
			return 0, err
		}
		data := temp[:total]
		if !ur.readBOM {
			ur.readBOM = true
			if len(data) >= 2 {
				if data[0] == 0xFF && data[1] == 0xFE {
					ur.isLE = true
					data = data[2:]
				} else if data[0] == 0xFE && data[1] == 0xFF {
					ur.isLE = false
					data = data[2:]
				} else {
					ur.isLE = true
				}
			} else {
				ur.isLE = true
			}
		}
		if len(data)%2 != 0 {
			ur.leftover = []byte{data[len(data)-1]}
			data = data[:len(data)-1]
		}
		if len(data) == 0 {
			if err != nil {
				return 0, err
			}
			return 0, nil
		}
		u16s := make([]uint16, len(data)/2)
		for i := 0; i < len(u16s); i++ {
			b1 := data[2*i]
			b2 := data[2*i+1]
			if ur.isLE {
				u16s[i] = uint16(b1) | (uint16(b2) << 8)
			} else {
				u16s[i] = uint16(b2) | (uint16(b1) << 8)
			}
		}
		runes := utf16.Decode(u16s)
		var utf8Buf []byte
		for _, r := range runes {
			var rBuf [4]byte
			wn := utf8.EncodeRune(rBuf[:], r)
			utf8Buf = append(utf8Buf, rBuf[:wn]...)
		}
		ur.buf = utf8Buf
	}
	n := copy(p, ur.buf)
	ur.buf = ur.buf[n:]
	return n, nil
}

// Convert reads CSV from r, parses and writes it as JSON to w.
func Convert(r io.Reader, w io.Writer, opts Options) error {
	var decoded io.Reader = r
	enc := strings.ToLower(opts.Encoding)
	if enc == "iso-8859-1" || enc == "iso88591" {
		decoded = &isoReader{r: r}
	} else if enc == "utf-16" || enc == "utf16" {
		decoded = &utf16Reader{r: r}
	}

	reader := csv.NewReader(decoded)
	if opts.Delimiter != 0 {
		reader.Comma = opts.Delimiter
	}
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			_, errWrite := w.Write([]byte("[]"))
			return errWrite
		}
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	headerCount := len(headers)
	if headerCount == 0 {
		_, errWrite := w.Write([]byte("[]"))
		return errWrite
	}

	if opts.Pretty {
		_, err = w.Write([]byte("[\n"))
	} else {
		_, err = w.Write([]byte("["))
	}
	if err != nil {
		return err
	}

	isFirst := true
	lineNum := 1

	for {
		lineNum++
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if opts.AbortOnError {
				return fmt.Errorf("malformed CSV syntax at line %d: %w", lineNum, err)
			}
			fmt.Fprintf(os.Stderr, "WARNING: Malformed CSV syntax at line %d: %v\n", lineNum, err)
			continue
		}

		if len(row) != headerCount {
			errMsg := fmt.Errorf("row %d has %d columns, expected %d", lineNum, len(row), headerCount)
			if opts.AbortOnError {
				return errMsg
			}
			fmt.Fprintf(os.Stderr, "WARNING: Mismatched column count at line %d: %v\n", lineNum, errMsg)
			continue
		}

		raw, err := marshalObject(headers, row, opts)
		if err != nil {
			return err
		}

		if !isFirst {
			if opts.Pretty {
				_, _ = w.Write([]byte(",\n"))
			} else {
				_, _ = w.Write([]byte(","))
			}
		}
		isFirst = false

		if opts.Pretty {
			var formatted bytes.Buffer
			_ = json.Indent(&formatted, raw, "  ", "  ")
			_, _ = w.Write([]byte("  "))
			_, _ = w.Write(formatted.Bytes())
		} else {
			_, _ = w.Write(raw)
		}
	}

	if opts.Pretty {
		_, err = w.Write([]byte("\n]\n"))
	} else {
		_, err = w.Write([]byte("]"))
	}
	return err
}

func marshalObject(headers []string, row []string, opts Options) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString("{")
	first := true
	for i, h := range headers {
		if i >= len(row) {
			break
		}
		val := row[i]
		if val == "" {
			if opts.EmptyStrategy == StrategyOmit {
				continue
			}
		}

		if !first {
			sb.WriteString(",")
		}
		first = false

		keyEscaped, _ := json.Marshal(h)
		sb.Write(keyEscaped)
		sb.WriteString(":")

		if val == "" {
			if opts.EmptyStrategy == StrategyNull {
				sb.WriteString("null")
			} else {
				sb.WriteString(`""`)
			}
			continue
		}

		var finalVal interface{}
		if opts.InferTypes {
			finalVal = inferType(val)
		} else {
			finalVal = val
		}

		if finalVal == nil {
			sb.WriteString("null")
		} else {
			valEscaped, err := json.Marshal(finalVal)
			if err != nil {
				return nil, err
			}
			sb.Write(valEscaped)
		}
	}
	sb.WriteString("}")
	return []byte(sb.String()), nil
}

func inferType(val string) interface{} {
	if val == "" {
		return nil
	}
	valLower := strings.ToLower(val)
	if valLower == "null" || valLower == "nil" {
		return nil
	}
	if valLower == "true" {
		return true
	}
	if valLower == "false" {
		return false
	}
	if i, err := strconv.ParseInt(val, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return f
	}
	return val
}

// RunCLI handles parsing CLI arguments and runs the converter.
// It returns an exit code matching the technical specifications.
func RunCLI(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("csv2json", flag.ContinueOnError)
	flags.SetOutput(stderr)

	inputPath := flags.String("input", "", "Source file path")
	flags.StringVar(inputPath, "i", "", "Source file path (shorthand)")

	outputPath := flags.String("output", "", "Destination file path")
	flags.StringVar(outputPath, "o", "", "Destination file path (shorthand)")

	pretty := flags.Bool("pretty", false, "Pretty print JSON output")

	delimStr := flags.String("delimiter", ",", "CSV delimiter character")
	flags.StringVar(delimStr, "d", ",", "CSV delimiter character (shorthand)")

	emptyStr := flags.String("empty-strategy", "null", "Empty value strategy (omit, null, empty)")
	flags.StringVar(emptyStr, "e", "null", "Empty value strategy (shorthand)")

	inferTypes := flags.Bool("infer-types", true, "Infer primitive data types")
	flags.BoolVar(inferTypes, "n", true, "Infer primitive data types (shorthand)")

	encoding := flags.String("encoding", "utf-8", "Input file encoding (utf-8, iso-8859-1, utf-16)")
	flags.StringVar(encoding, "c", "utf-8", "Input file encoding (shorthand)")

	abortOnError := flags.Bool("abort-on-error", false, "Abort processing on malformed rows")
	flags.BoolVar(abortOnError, "a", false, "Abort processing on malformed rows (shorthand)")

	err := flags.Parse(args)
	if err != nil {
		return 2
	}

	var delimiter rune
	if *delimStr != "" {
		runes := []rune(*delimStr)
		if len(runes) != 1 {
			fmt.Fprintln(stderr, "Error: Delimiter must be a single character")
			return 2
		}
		delimiter = runes[0]
	}

	strategy := EmptyStrategy(*emptyStr)
	if strategy != StrategyOmit && strategy != StrategyNull && strategy != StrategyEmpty {
		fmt.Fprintln(stderr, "Error: Invalid empty strategy, must be one of: omit, null, empty")
		return 2
	}

	var in io.Reader
	if *inputPath == "" {
		in = stdin
	} else {
		file, err := os.Open(*inputPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(stderr, "Error: Input file not found: %s\n", *inputPath)
				return 1
			}
			if os.IsPermission(err) {
				fmt.Fprintf(stderr, "Error: Permission denied for input file: %s\n", *inputPath)
				return 1
			}
			fmt.Fprintf(stderr, "Error opening input file: %v\n", err)
			return 1
		}
		defer file.Close()
		in = file
	}

	var out io.Writer
	if *outputPath == "" {
		out = stdout
	} else {
		file, err := os.Create(*outputPath)
		if err != nil {
			if os.IsPermission(err) {
				fmt.Fprintf(stderr, "Error: Permission denied for output file: %s\n", *outputPath)
				return 1
			}
			fmt.Fprintf(stderr, "Error creating output file: %v\n", err)
			return 3
		}
		defer file.Close()
		out = file
	}

	opts := Options{
		Delimiter:     delimiter,
		InferTypes:    *inferTypes,
		EmptyStrategy: strategy,
		AbortOnError:  *abortOnError,
		Pretty:        *pretty,
		Encoding:      *encoding,
	}

	err = Convert(in, out, opts)
	if err != nil {
		fmt.Fprintf(stderr, "Conversion error: %v\n", err)
		if strings.Contains(err.Error(), "malformed") || strings.Contains(err.Error(), "column") {
			return 2
		}
		return 3
	}

	return 0
}
