package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunFormatsStdin(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run(nil, strings.NewReader(`{"a":1,"b":[2,3]}`), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if want := "{ \"a\": 1, \"b\": [2, 3] }\n"; stdout.String() != want {
		t.Errorf("got %q, want %q", stdout.String(), want)
	}
}

// TestRunFlags checks that every flag reaches the formatter, in both its short
// and its long spelling.
func TestRunFlags(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		stdin string
		want  string
	}{
		{"no flags", nil, `{"a":1,"b":[2,3]}`, "{ \"a\": 1, \"b\": [2, 3] }\n"},
		{"explicit stdin", []string{"-"}, `{"a":1}`, "{ \"a\": 1 }\n"},

		{"indent short", []string{"-i", "2"}, `{"o":{"a":1}}`, "{\n  \"o\": { \"a\": 1 }\n}\n"},
		{"indent long", []string{"--indent=2"}, `{"o":{"a":1}}`, "{\n  \"o\": { \"a\": 1 }\n}\n"},
		{"indent zero", []string{"-i", "0"}, `{"o":{"a":1}}`, "{\n\"o\": { \"a\": 1 }\n}\n"},

		{"tab short", []string{"-t"}, `{"o":{"a":1}}`, "{\n\t\"o\": { \"a\": 1 }\n}\n"},
		{"tab long", []string{"--tab"}, `{"o":{"a":1}}`, "{\n\t\"o\": { \"a\": 1 }\n}\n"},
		{"tab beats indent", []string{"-i", "2", "-t"}, `{"o":{"a":1}}`, "{\n\t\"o\": { \"a\": 1 }\n}\n"},

		{"max width short", []string{"-x", "20"}, `{"a":1,"b":[2,3]}`, "{\n    \"a\": 1,\n    \"b\": [2, 3]\n}\n"},
		{"max width long", []string{"--max-width=20"}, `{"a":1,"b":[2,3]}`, "{\n    \"a\": 1,\n    \"b\": [2, 3]\n}\n"},
		{"max width zero disables", []string{"-x", "0"}, `{"a":1,"b":[2,3]}`, "{ \"a\": 1, \"b\": [2, 3] }\n"},

		{"max items short", []string{"-n", "1"}, `{"a":1,"b":[2,3]}`, "{\n    \"a\": 1,\n    \"b\": [\n        2,\n        3\n    ]\n}\n"},
		{"max items long", []string{"--max-items=1"}, `{"a":1,"b":[2,3]}`, "{\n    \"a\": 1,\n    \"b\": [\n        2,\n        3\n    ]\n}\n"},
		{"max items zero disables", []string{"-n", "0"}, `{"a":1,"b":[2,3]}`, "{ \"a\": 1, \"b\": [2, 3] }\n"},

		{"tight braces short", []string{"-b"}, `{"a":1}`, "{\"a\": 1}\n"},
		{"tight braces long", []string{"--tight-braces"}, `{"a":1}`, "{\"a\": 1}\n"},

		{"padded brackets short", []string{"-p"}, `{"a":[1]}`, "{ \"a\": [ 1 ] }\n"},
		{"padded brackets long", []string{"--padded-brackets"}, `{"a":[1]}`, "{ \"a\": [ 1 ] }\n"},

		{"sort keys short", []string{"-s"}, `{"b":1,"a":2}`, "{ \"a\": 2, \"b\": 1 }\n"},
		{"sort keys long", []string{"--sort-keys"}, `{"b":1,"a":2}`, "{ \"a\": 2, \"b\": 1 }\n"},

		{"flags combine", []string{"-s", "-b", "-p", "-i", "2"}, `{"b":{"z":1,"y":2},"a":[1]}`,
			"{\n  \"a\": [ 1 ],\n  \"b\": {\"y\": 2, \"z\": 1}\n}\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tc.args, strings.NewReader(tc.stdin), &stdout, &stderr)
			if code != 0 {
				t.Fatalf("exit %d, stderr: %s", code, stderr.String())
			}
			if stdout.String() != tc.want {
				t.Errorf("got\n%q\nwant\n%q", stdout.String(), tc.want)
			}
		})
	}
}

func TestRunPrintsVersion(t *testing.T) {
	for _, arg := range []string{"-v", "--version"} {
		t.Run(arg, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run([]string{arg}, strings.NewReader("not json"), &stdout, &stderr); code != 0 {
				t.Fatalf("exit %d, stderr: %s", code, stderr.String())
			}
			if want := "jrp " + version + "\n"; stdout.String() != want {
				t.Errorf("got %q, want %q", stdout.String(), want)
			}
		})
	}
}

func TestRunPrintsHelp(t *testing.T) {
	for _, arg := range []string{"-h", "--help"} {
		t.Run(arg, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run([]string{arg}, strings.NewReader(""), &stdout, &stderr); code != 0 {
				t.Errorf("asking for help is not a failure, got exit %d", code)
			}
			if !strings.Contains(stderr.String(), "Usage:") {
				t.Errorf("help should describe the usage, got %q", stderr.String())
			}
			if stdout.Len() != 0 {
				t.Errorf("help should not write to stdout, got %q", stdout.String())
			}
		})
	}
}

func TestRunRejectsBadFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{"unknown flag", []string{"--nope"}, "flag provided but not defined"},
		{"non numeric indent", []string{"-i", "wide"}, "invalid value"},
		{"negative indent", []string{"-i", "-1"}, "cannot be negative"},
		{"negative max width", []string{"-x", "-1"}, "cannot be negative"},
		{"negative max items", []string{"-n", "-1"}, "cannot be negative"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(tc.args, strings.NewReader(`{"a":1}`), &stdout, &stderr); code != 1 {
				t.Errorf("exit %d, want 1", code)
			}
			if !strings.Contains(stderr.String(), tc.wantMsg) {
				t.Errorf("got %q, want it to contain %q", stderr.String(), tc.wantMsg)
			}
			if stdout.Len() != 0 {
				t.Errorf("nothing should have been written, got %q", stdout.String())
			}
		})
	}
}

func TestRunFormatsSeveralFilesToStdout(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.json")
	second := filepath.Join(dir, "second.json")
	for path, body := range map[string]string{first: `{"a":1}`, second: `[2,3]`} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{first, second}, strings.NewReader(""), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if want := "{ \"a\": 1 }\n[2, 3]\n"; stdout.String() != want {
		t.Errorf("got %q, want %q", stdout.String(), want)
	}
	// Without -w the inputs must be untouched.
	if body, err := os.ReadFile(first); err != nil || string(body) != `{"a":1}` {
		t.Errorf("input was modified: %q (%v)", body, err)
	}
}

func TestRunKeepsGoingAfterAMissingFile(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.json")
	if err := os.WriteFile(good, []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing.json")

	var stdout, stderr bytes.Buffer
	code := run([]string{missing, good}, strings.NewReader(""), &stdout, &stderr)

	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "missing.json") {
		t.Errorf("error should name the file, got %q", stderr.String())
	}
	if want := "{ \"a\": 1 }\n"; stdout.String() != want {
		t.Errorf("the readable file should still be formatted, got %q", stdout.String())
	}
}

func TestRunWritesSeveralFilesInPlace(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		filepath.Join(dir, "a.json"): `{"a":1}`,
		filepath.Join(dir, "b.json"): `{"o":{"b":2}}`,
	}
	var args []string
	for path, body := range files {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		args = append(args, path)
	}

	var stdout, stderr bytes.Buffer
	if code := run(append([]string{"--write"}, args...), strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}

	want := map[string]string{
		filepath.Join(dir, "a.json"): "{ \"a\": 1 }\n",
		filepath.Join(dir, "b.json"): "{\n    \"o\": { \"b\": 2 }\n}\n",
	}
	for path, body := range want {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != body {
			t.Errorf("%s: got %q, want %q", filepath.Base(path), got, body)
		}
	}
}

func TestRunWriteIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(path, []byte(`{"o":{"a":1},"b":[1,2]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var first []byte
	for pass := range 2 {
		var stdout, stderr bytes.Buffer
		if code := run([]string{"-w", path}, strings.NewReader(""), &stdout, &stderr); code != 0 {
			t.Fatalf("pass %d: exit %d, stderr: %s", pass, code, stderr.String())
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if pass == 0 {
			first = got
		} else if !bytes.Equal(first, got) {
			t.Errorf("second pass changed the file\n--- first ---\n%s\n--- second ---\n%s", first, got)
		}
	}
}

func TestRunWritesInPlace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(path, []byte(`{"o":{"a":1},"b":2}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-w", path}, strings.NewReader(""), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("-w should not write to stdout, got %q", stdout.String())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n    \"o\": { \"a\": 1 },\n    \"b\": 2\n}\n"
	if string(got) != want {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}

	// The temporary file used for the swap must not be left behind.
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("expected only the input file to remain, got %d entries", len(entries))
	}
}

func TestRunRejectsWriteToStdin(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"-w"}, strings.NewReader(`{"a":1}`), &stdout, &stderr)

	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "cannot rewrite stdin") {
		t.Errorf("unhelpful error: %q", stderr.String())
	}
}

func TestRunReportsBadInputWithoutTouchingTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.json")
	const original = `{"a":`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-w", path}, strings.NewReader(""), &stdout, &stderr)

	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "broken.json") {
		t.Errorf("error should name the file, got %q", stderr.String())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Errorf("file was modified despite the error: %q", got)
	}
}
