package pipeline

import "testing"

func TestContainPath(t *testing.T) {
	target := "workspace/password"
	cases := []struct {
		raw     string
		want    string
		wantOK  bool
	}{
		{"workspace/password/hash.go", "workspace/password/hash.go", true},
		{"/workspace/password/hash.go", "workspace/password/hash.go", true}, // leading slash tolerated
		{"workspace/password/sub/x.go", "workspace/password/sub/x.go", true},
		{"workspace/other/evil.go", "", false},        // sibling folder
		{"workspace/password/../other/x.go", "", false}, // traversal
		{"/etc/passwd", "", false},                      // absolute escape
		{"../../secrets.go", "", false},                 // relative escape
	}
	for _, c := range cases {
		got, ok := containPath(target, c.raw)
		if ok != c.wantOK || got != c.want {
			t.Errorf("containPath(%q, %q) = (%q, %v), want (%q, %v)", target, c.raw, got, ok, c.want, c.wantOK)
		}
	}
}

func TestParseDoDTarget(t *testing.T) {
	cases := []struct {
		name           string
		dod            string
		wantTarget     string
		wantFeature    string
	}{
		{"explicit target", "# TASK: pw\n- Target Subfolder: workspace/pw\n", "workspace/pw", "pw"},
		{"task only", "# TASK: fib\n", "workspace/fib", "fib"},
		{"empty", "", "workspace/default_task", "default_task"},
	}
	for _, c := range cases {
		gt, gf := parseDoDTarget(c.dod)
		if gt != c.wantTarget || gf != c.wantFeature {
			t.Errorf("%s: parseDoDTarget = (%q,%q), want (%q,%q)", c.name, gt, gf, c.wantTarget, c.wantFeature)
		}
	}
}
