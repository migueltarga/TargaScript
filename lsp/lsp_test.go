package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/diag"
)

func frame(v any) string {
	body, _ := json.Marshal(v)
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

// replies runs a session and returns the messages the server sent.
func replies(t *testing.T, check Checker, messages ...any) []map[string]any {
	t.Helper()
	var in strings.Builder
	for _, m := range messages {
		in.WriteString(frame(m))
	}
	var out bytes.Buffer
	if err := NewServer(&out, check).Serve(strings.NewReader(in.String())); err != nil {
		t.Fatal(err)
	}
	sent := []map[string]any{}
	reader := bufio.NewReader(&out)
	for {
		body, err := readMessage(reader)
		if err != nil {
			break
		}
		var m map[string]any
		json.Unmarshal(body, &m)
		sent = append(sent, m)
	}
	return sent
}

func TestDiagnosticsCompletionAndHover(t *testing.T) {
	check := func(source, dir string) []diag.Diagnostic {
		if dir != "/home/ana" {
			t.Errorf("checked in %q, want /home/ana", dir)
		}
		if strings.Contains(source, "totl") {
			return []diag.Diagnostic{{Line: 2, Column: 7, Message: `"totl" is not defined`}}
		}
		return nil
	}
	uri := "file:///home/ana/main.tg"
	sent := replies(t, check,
		map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}},
		map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": "let total = 1\nprint(totl)"}}},
		map[string]any{"jsonrpc": "2.0", "method": "textDocument/didChange", "params": map[string]any{
			"textDocument": map[string]any{"uri": uri}, "contentChanges": []any{map[string]any{"text": "let x = fetch(1)"}}}},
		map[string]any{"jsonrpc": "2.0", "id": 2, "method": "textDocument/completion", "params": map[string]any{}},
		map[string]any{"jsonrpc": "2.0", "id": 3, "method": "textDocument/hover", "params": map[string]any{
			"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 0, "character": 10}}},
		map[string]any{"jsonrpc": "2.0", "id": 4, "method": "workspace/symbol", "params": map[string]any{}},
		map[string]any{"jsonrpc": "2.0", "id": 5, "method": "shutdown"},
		map[string]any{"jsonrpc": "2.0", "method": "exit"},
	)

	if len(sent) != 7 {
		t.Fatalf("sent %d messages: %v", len(sent), sent)
	}
	capabilities := sent[0]["result"].(map[string]any)["capabilities"].(map[string]any)
	if capabilities["textDocumentSync"].(float64) != 1 || capabilities["hoverProvider"] != true {
		t.Errorf("capabilities: %v", capabilities)
	}

	opened := sent[1]["params"].(map[string]any)
	problems := opened["diagnostics"].([]any)
	if sent[1]["method"] != "textDocument/publishDiagnostics" || len(problems) != 1 {
		t.Fatalf("after didOpen: %v", sent[1])
	}
	start := problems[0].(map[string]any)["range"].(map[string]any)["start"].(map[string]any)
	if start["line"].(float64) != 1 || start["character"].(float64) != 6 {
		t.Errorf("the problem starts at %v, want line 1, character 6", start)
	}
	if changed := sent[2]["params"].(map[string]any)["diagnostics"].([]any); len(changed) != 0 {
		t.Errorf("after the fix: %v", changed)
	}

	items := sent[3]["result"].([]any)
	found := false
	for _, item := range items {
		if item.(map[string]any)["label"] == "parseJSON" && item.(map[string]any)["detail"] == "parseJSON(text)" {
			found = true
		}
	}
	if !found {
		t.Errorf("parseJSON isn't completed with its signature")
	}

	hover := sent[4]["result"].(map[string]any)["contents"].(map[string]any)["value"].(string)
	if !strings.Contains(hover, "fetch(url, options?)") {
		t.Errorf("hover over fetch: %q", hover)
	}
	if sent[5]["result"] != nil || sent[6]["result"] != nil {
		t.Errorf("unknown requests and shutdown should answer null: %v %v", sent[5], sent[6])
	}
}

func TestDefinitionsAndSymbols(t *testing.T) {
	source := "fn add(a, b) { a + b }\nlet total = add(1, 2)\nfn outer() {\n  let total = 5\n  total + add(0, 0)\n}\nconst LIMIT = 10\nprint(total, LIMIT)\n"
	uri := "file:///x/main.tg"
	none := func(string, string) []diag.Diagnostic { return nil }
	at := func(line, character int) map[string]any {
		return map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": character}}
	}
	sent := replies(t, none,
		map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": source}}},
		// add in line 2 goes to its fn
		map[string]any{"jsonrpc": "2.0", "id": 1, "method": "textDocument/definition", "params": at(1, 13)},
		// total inside outer goes to the closer let, in line 4
		map[string]any{"jsonrpc": "2.0", "id": 2, "method": "textDocument/definition", "params": at(4, 3)},
		// total in the last line goes to the last let before it
		map[string]any{"jsonrpc": "2.0", "id": 3, "method": "textDocument/definition", "params": at(7, 7)},
		map[string]any{"jsonrpc": "2.0", "id": 4, "method": "textDocument/documentSymbol", "params": map[string]any{
			"textDocument": map[string]any{"uri": uri}}},
		map[string]any{"jsonrpc": "2.0", "id": 5, "method": "textDocument/completion", "params": map[string]any{
			"textDocument": map[string]any{"uri": uri}}},
		map[string]any{"jsonrpc": "2.0", "method": "exit"},
	)

	startOf := func(m map[string]any) [2]float64 {
		start := m["result"].(map[string]any)["range"].(map[string]any)["start"].(map[string]any)
		return [2]float64{start["line"].(float64), start["character"].(float64)}
	}
	if got := startOf(sent[1]); got != [2]float64{0, 3} {
		t.Errorf("add goes to %v, want line 0, character 3", got)
	}
	if got := startOf(sent[2]); got != [2]float64{3, 6} {
		t.Errorf("the inner total goes to %v, want line 3, character 6", got)
	}
	// outer's total is in a block that's closed there
	if got := startOf(sent[3]); got != [2]float64{1, 4} {
		t.Errorf("the last total goes to %v, want the top-level let, line 1", got)
	}

	names := []string{}
	for _, symbol := range sent[4]["result"].([]any) {
		names = append(names, symbol.(map[string]any)["name"].(string))
	}
	if strings.Join(names, " ") != "add total outer LIMIT" {
		t.Errorf("outline: %v", names)
	}

	completed := map[string]bool{}
	for _, item := range sent[5]["result"].([]any) {
		completed[item.(map[string]any)["label"].(string)] = true
	}
	if !completed["outer"] || !completed["LIMIT"] || !completed["fetch"] || !completed["repeat"] {
		t.Errorf("completion misses the file's names, builtins or keywords")
	}
}

func TestFormatting(t *testing.T) {
	uri := "file:///x/messy.tg"
	none := func(string, string) []diag.Diagnostic { return nil }
	sent := replies(t, none,
		map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": "fn f() {\nreturn 1   \n}"}}},
		map[string]any{"jsonrpc": "2.0", "id": 1, "method": "textDocument/formatting", "params": map[string]any{
			"textDocument": map[string]any{"uri": uri}}},
		map[string]any{"jsonrpc": "2.0", "method": "exit"},
	)
	edits := sent[1]["result"].([]any)
	if len(edits) != 1 {
		t.Fatalf("edits: %v", edits)
	}
	edit := edits[0].(map[string]any)
	end := edit["range"].(map[string]any)["end"].(map[string]any)
	if edit["newText"] != "fn f() {\n    return 1\n}\n" || end["line"].(float64) != 2 || end["character"].(float64) != 1 {
		t.Errorf("edit: %v", edit)
	}
}

// Structs and enums show in the outline with their own kinds
func TestStructAndEnumSymbols(t *testing.T) {
	got := []string{}
	for _, d := range declarations("struct Point { x, y }\nenum Status { On, Off }\nlet struct = 1\n") {
		got = append(got, d.name+":"+strconv.Itoa(d.kind))
	}
	if strings.Join(got, " ") != "Point:23 Status:10 struct:13" {
		t.Errorf("got %v", got)
	}
}
