package csv_to_json

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertBasic(t *testing.T) {
	input := "name,age,active\nAlice,30,true\nBob,25,false\n"
	expected := `[{"name":"Alice","age":30,"active":true},{"name":"Bob","age":25,"active":false}]`

	var out bytes.Buffer
	err := Convert(strings.NewReader(input), &out, Options{
		Delimiter:     ',',
		InferTypes:    true,
		EmptyStrategy: StrategyNull,
		Encoding:      "utf-8",
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	if out.String() != expected {
		t.Errorf("expected %s, got %s", expected, out.String())
	}
}

func TestConvertPretty(t *testing.T) {
	input := "name,age\nAlice,30\n"
	expected := "[\n  {\n    \"name\": \"Alice\",\n    \"age\": 30\n  }\n]\n"

	var out bytes.Buffer
	err := Convert(strings.NewReader(input), &out, Options{
		Delimiter:     ',',
		InferTypes:    true,
		EmptyStrategy: StrategyNull,
		Pretty:        true,
		Encoding:      "utf-8",
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	if out.String() != expected {
		t.Errorf("expected %q, got %q", expected, out.String())
	}
}

func TestCustomDelimiter(t *testing.T) {
	tests := []struct {
		delim rune
		input string
	}{
		{';', "name;age\nAlice;30\n"},
		{'\t', "name\tage\nAlice\t30\n"},
		{'|', "name|age\nAlice|30\n"},
	}

	for _, tc := range tests {
		var out bytes.Buffer
		err := Convert(strings.NewReader(tc.input), &out, Options{
			Delimiter:     tc.delim,
			InferTypes:    true,
			EmptyStrategy: StrategyNull,
			Encoding:      "utf-8",
		})
		if err != nil {
			t.Fatalf("Convert with delimiter %c returned error: %v", tc.delim, err)
		}
		expected := `[{"name":"Alice","age":30}]`
		if out.String() != expected {
			t.Errorf("expected %s, got %s", expected, out.String())
		}
	}
}

func TestTypeInference(t *testing.T) {
	input := "int_val,float_val,bool_val,null_val,string_val\n123,45.67,true,null,hello\n"
	expected := `[{"int_val":123,"float_val":45.67,"bool_val":true,"null_val":null,"string_val":"hello"}]`

	var out bytes.Buffer
	err := Convert(strings.NewReader(input), &out, Options{
		Delimiter:     ',',
		InferTypes:    true,
		EmptyStrategy: StrategyNull,
		Encoding:      "utf-8",
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	if out.String() != expected {
		t.Errorf("expected %s, got %s", expected, out.String())
	}

	// Without type inference
	var out2 bytes.Buffer
	err = Convert(strings.NewReader(input), &out2, Options{
		Delimiter:     ',',
		InferTypes:    false,
		EmptyStrategy: StrategyNull,
		Encoding:      "utf-8",
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	expectedNoInfer := `[{"int_val":"123","float_val":"45.67","bool_val":"true","null_val":"null","string_val":"hello"}]`
	if out2.String() != expectedNoInfer {
		t.Errorf("expected %s, got %s", expectedNoInfer, out2.String())
	}
}

func TestEmptyValueStrategies(t *testing.T) {
	input := "name,age,city\nAlice,,Paris\n"

	// Omit
	var outOmit bytes.Buffer
	_ = Convert(strings.NewReader(input), &outOmit, Options{
		Delimiter:     ',',
		InferTypes:    true,
		EmptyStrategy: StrategyOmit,
		Encoding:      "utf-8",
	})
	expectedOmit := `[{"name":"Alice","city":"Paris"}]`
	if outOmit.String() != expectedOmit {
		t.Errorf("StrategyOmit expected %s, got %s", expectedOmit, outOmit.String())
	}

	// Null
	var outNull bytes.Buffer
	_ = Convert(strings.NewReader(input), &outNull, Options{
		Delimiter:     ',',
		InferTypes:    true,
		EmptyStrategy: StrategyNull,
		Encoding:      "utf-8",
	})
	expectedNull := `[{"name":"Alice","age":null,"city":"Paris"}]`
	if outNull.String() != expectedNull {
		t.Errorf("StrategyNull expected %s, got %s", expectedNull, outNull.String())
	}

	// Empty string
	var outEmpty bytes.Buffer
	_ = Convert(strings.NewReader(input), &outEmpty, Options{
		Delimiter:     ',',
		InferTypes:    true,
		EmptyStrategy: StrategyEmpty,
		Encoding:      "utf-8",
	})
	expectedEmpty := `[{"name":"Alice","age":"","city":"Paris"}]`
	if outEmpty.String() != expectedEmpty {
		t.Errorf("StrategyEmpty expected %s, got %s", expectedEmpty, outEmpty.String())
	}
}

func TestRFC4180Compliance(t *testing.T) {
	// Escaped double-quotes and multi-line fields
	input := `name,description,score
"Bob","He said, ""Hello!""",95
"Alice","Multi-line
field description",98
`
	expected := `[{"name":"Bob","description":"He said, \"Hello!\"","score":95},{"name":"Alice","description":"Multi-line\nfield description","score":98}]`

	var out bytes.Buffer
	err := Convert(strings.NewReader(input), &out, Options{
		Delimiter:     ',',
		InferTypes:    true,
		EmptyStrategy: StrategyNull,
		Encoding:      "utf-8",
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	if out.String() != expected {
		t.Errorf("expected %s, got %s", expected, out.String())
	}
}

func TestTrailingCommas(t *testing.T) {
	// Trailing commas in header and data
	input := "name,age,\nAlice,30,\n"
	expected := `[{"name":"Alice","age":30,"":null}]`

	var out bytes.Buffer
	err := Convert(strings.NewReader(input), &out, Options{
		Delimiter:     ',',
		InferTypes:    true,
		EmptyStrategy: StrategyNull,
		Encoding:      "utf-8",
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	if out.String() != expected {
		t.Errorf("expected %s, got %s", expected, out.String())
	}
}

func TestSpecialCharactersInHeaders(t *testing.T) {
	input := "first-name,age@years,#id\nAlice,30,1\n"
	expected := `[{"first-name":"Alice","age@years":30,"#id":1}]`

	var out bytes.Buffer
	err := Convert(strings.NewReader(input), &out, Options{
		Delimiter:     ',',
		InferTypes:    true,
		EmptyStrategy: StrategyNull,
		Encoding:      "utf-8",
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	if out.String() != expected {
		t.Errorf("expected %s, got %s", expected, out.String())
	}
}

func TestEmptyFile(t *testing.T) {
	var out bytes.Buffer
	err := Convert(strings.NewReader(""), &out, Options{
		Delimiter:     ',',
		InferTypes:    true,
		EmptyStrategy: StrategyNull,
		Encoding:      "utf-8",
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if out.String() != "[]" {
		t.Errorf("expected [], got %s", out.String())
	}
}

func TestEncodingISO8859_1(t *testing.T) {
	// ISO-8859-1 encoded string: "name,value\näöü,123\n"
	// ä = 0xE4, ö = 0xF6, ü = 0xFC
	isoData := []byte{
		'n', 'a', 'm', 'e', ',', 'v', 'a', 'l', 'u', 'e', '\n',
		0xE4, 0xF6, 0xFC, ',', '1', '2', '3', '\n',
	}

	var out bytes.Buffer
	err := Convert(bytes.NewReader(isoData), &out, Options{
		Delimiter:     ',',
		InferTypes:    true,
		EmptyStrategy: StrategyNull,
		Encoding:      "iso-8859-1",
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	expected := `[{"name":"äöü","value":123}]`
	if out.String() != expected {
		t.Errorf("expected %s, got %s", expected, out.String())
	}
}

func TestEncodingUTF16(t *testing.T) {
	// UTF-16LE with BOM
	bomLE := []byte{0xFF, 0xFE}
	rawStr := "name,value\nAlice,123\n"
	var u16LE []byte
	u16LE = append(u16LE, bomLE...)
	for _, r := range rawStr {
		u16LE = append(u16LE, byte(r), 0)
	}

	var outLE bytes.Buffer
	err := Convert(bytes.NewReader(u16LE), &outLE, Options{
		Delimiter:     ',',
		InferTypes:    true,
		EmptyStrategy: StrategyNull,
		Encoding:      "utf-16",
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	expected := `[{"name":"Alice","value":123}]`
	if outLE.String() != expected {
		t.Errorf("LE expected %s, got %s", expected, outLE.String())
	}

	// UTF-16BE with BOM
	bomBE := []byte{0xFE, 0xFF}
	var u16BE []byte
	u16BE = append(u16BE, bomBE...)
	for _, r := range rawStr {
		u16BE = append(u16BE, 0, byte(r))
	}

	var outBE bytes.Buffer
	err = Convert(bytes.NewReader(u16BE), &outBE, Options{
		Delimiter:     ',',
		InferTypes:    true,
		EmptyStrategy: StrategyNull,
		Encoding:      "utf-16",
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	if outBE.String() != expected {
		t.Errorf("BE expected %s, got %s", expected, outBE.String())
	}
}

func TestMalformedRows(t *testing.T) {
	// Skip on mismatched columns
	input := "name,age\nAlice,30\nBob\nCharlie,28\n"
	expectedSkip := `[{"name":"Alice","age":30},{"name":"Charlie","age":28}]`

	var outSkip bytes.Buffer
	err := Convert(strings.NewReader(input), &outSkip, Options{
		Delimiter:    ',',
		InferTypes:   true,
		AbortOnError: false,
	})
	if err != nil {
		t.Fatalf("Convert with AbortOnError=false returned error: %v", err)
	}
	if outSkip.String() != expectedSkip {
		t.Errorf("expected %s, got %s", expectedSkip, outSkip.String())
	}

	// Abort on mismatched columns
	var outAbort bytes.Buffer
	err = Convert(strings.NewReader(input), &outAbort, Options{
		Delimiter:    ',',
		InferTypes:   true,
		AbortOnError: true,
	})
	if err == nil {
		t.Error("Convert with AbortOnError=true should have failed")
	}
}

func TestRunCLIBasic(t *testing.T) {
	tmpDir := t.TempDir()

	csvPath := filepath.Join(tmpDir, "test.csv")
	jsonPath := filepath.Join(tmpDir, "test.json")

	err := os.WriteFile(csvPath, []byte("name,age\nAlice,30\n"), 0644)
	if err != nil {
		t.Fatalf("failed to write test csv file: %v", err)
	}

	var stderr bytes.Buffer
	var stdout bytes.Buffer

	args := []string{"-i", csvPath, "-o", jsonPath}
	exitCode := RunCLI(args, nil, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("RunCLI failed with exit code %d, stderr: %s", exitCode, stderr.String())
	}

	jsonBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read output json file: %v", err)
	}

	expected := `[{"name":"Alice","age":30}]`
	if string(jsonBytes) != expected {
		t.Errorf("expected %s, got %s", expected, string(jsonBytes))
	}
}

func TestRunCLIErrors(t *testing.T) {
	// File not found error -> exit code 1
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	exitCode := RunCLI([]string{"-i", "non_existent_file.csv"}, nil, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("expected exit code 1 for non-existent file, got %d", exitCode)
	}

	// Invalid flags -> exit code 2
	stderr.Reset()
	exitCode = RunCLI([]string{"--invalid-flag"}, nil, &stdout, &stderr)
	if exitCode != 2 {
		t.Errorf("expected exit code 2 for invalid flag, got %d", exitCode)
	}

	// Invalid delimiter -> exit code 2
	stderr.Reset()
	exitCode = RunCLI([]string{"-d", "invalid_delim"}, nil, &stdout, &stderr)
	if exitCode != 2 {
		t.Errorf("expected exit code 2 for multi-char delimiter, got %d", exitCode)
	}

	// Invalid empty strategy -> exit code 2
	stderr.Reset()
	exitCode = RunCLI([]string{"-e", "invalid_strategy"}, nil, &stdout, &stderr)
	if exitCode != 2 {
		t.Errorf("expected exit code 2 for invalid empty strategy, got %d", exitCode)
	}
}
