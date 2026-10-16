// Package lsp is a Language Server Protocol server for TargaScript, so
// editors like Neovim, Helix and Zed show its problems as code is typed and
// complete names. It talks JSON-RPC over standard input and output.
package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/migueltarga/TargaScript/diag"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/format"
)

// Checker finds the problems in a program, like repl.Check.
type Checker func(source, dir string) []diag.Diagnostic

// Server answers an editor's requests about the files it has open.
type Server struct {
	check Checker
	out   io.Writer
	files map[string]string // text of the open files, by URI
}

// NewServer returns a server that writes to out and checks files with
// check.
func NewServer(out io.Writer, check Checker) *Server {
	return &Server{check: check, out: out, files: map[string]string{}}
}

type message struct {
	ID     *json.RawMessage `json:"id,omitempty"`
	Method string           `json:"method,omitempty"`
	Params json.RawMessage  `json:"params,omitempty"`
}

// Serve reads messages from in until the editor exits or in ends.
func (s *Server) Serve(in io.Reader) error {
	reader := bufio.NewReader(in)
	for {
		body, err := readMessage(reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		var m message
		if err := json.Unmarshal(body, &m); err != nil {
			continue
		}
		if m.Method == "exit" {
			return nil
		}
		s.handle(m)
	}
}

// readMessage reads one message: headers, a blank line, then a body of
// Content-Length bytes.
func readMessage(r *bufio.Reader) ([]byte, error) {
	length := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if name, value, ok := strings.Cut(line, ":"); ok && strings.EqualFold(name, "Content-Length") {
			length, _ = strconv.Atoi(strings.TrimSpace(value))
		}
	}
	if length < 0 {
		return nil, fmt.Errorf("message without Content-Length")
	}
	body := make([]byte, length)
	_, err := io.ReadFull(r, body)
	return body, err
}

func (s *Server) send(v any) {
	body, _ := json.Marshal(v)
	fmt.Fprintf(s.out, "Content-Length: %d\r\n\r\n%s", len(body), body)
}

func (s *Server) reply(id *json.RawMessage, result any) {
	s.send(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func (s *Server) notify(method string, params any) {
	s.send(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

type textDocument struct {
	URI  string `json:"uri"`
	Text string `json:"text"`
}

func (s *Server) handle(m message) {
	switch m.Method {
	case "initialize":
		s.reply(m.ID, map[string]any{
			"capabilities": map[string]any{
				// The editor sends the whole text on every change
				"textDocumentSync":           1,
				"completionProvider":         map[string]any{},
				"hoverProvider":              true,
				"definitionProvider":         true,
				"documentSymbolProvider":     true,
				"documentFormattingProvider": true,
			},
			"serverInfo": map[string]any{"name": "targascript"},
		})
	case "shutdown":
		s.reply(m.ID, nil)
	case "textDocument/didOpen":
		var p struct {
			TextDocument textDocument `json:"textDocument"`
		}
		if json.Unmarshal(m.Params, &p) == nil {
			s.update(p.TextDocument.URI, p.TextDocument.Text)
		}
	case "textDocument/didChange":
		var p struct {
			TextDocument   textDocument `json:"textDocument"`
			ContentChanges []struct {
				Text string `json:"text"`
			} `json:"contentChanges"`
		}
		if json.Unmarshal(m.Params, &p) == nil && len(p.ContentChanges) > 0 {
			s.update(p.TextDocument.URI, p.ContentChanges[len(p.ContentChanges)-1].Text)
		}
	case "textDocument/didClose":
		var p struct {
			TextDocument textDocument `json:"textDocument"`
		}
		if json.Unmarshal(m.Params, &p) == nil {
			delete(s.files, p.TextDocument.URI)
			s.notify("textDocument/publishDiagnostics", map[string]any{"uri": p.TextDocument.URI, "diagnostics": []any{}})
		}
	case "textDocument/completion":
		var p struct {
			TextDocument textDocument `json:"textDocument"`
		}
		json.Unmarshal(m.Params, &p)
		s.reply(m.ID, s.completions(p.TextDocument.URI))
	case "textDocument/definition":
		var p struct {
			TextDocument textDocument `json:"textDocument"`
			Position     position     `json:"position"`
		}
		if json.Unmarshal(m.Params, &p) != nil {
			s.reply(m.ID, nil)
			return
		}
		s.reply(m.ID, s.definition(p.TextDocument.URI, p.Position))
	case "textDocument/formatting":
		var p struct {
			TextDocument textDocument `json:"textDocument"`
		}
		if json.Unmarshal(m.Params, &p) != nil {
			s.reply(m.ID, nil)
			return
		}
		s.reply(m.ID, s.format(p.TextDocument.URI))
	case "textDocument/documentSymbol":
		var p struct {
			TextDocument textDocument `json:"textDocument"`
		}
		if json.Unmarshal(m.Params, &p) != nil {
			s.reply(m.ID, nil)
			return
		}
		s.reply(m.ID, s.symbols(p.TextDocument.URI))
	case "textDocument/hover":
		var p struct {
			TextDocument textDocument `json:"textDocument"`
			Position     position     `json:"position"`
		}
		if json.Unmarshal(m.Params, &p) != nil {
			s.reply(m.ID, nil)
			return
		}
		s.reply(m.ID, s.hover(p.TextDocument.URI, p.Position))
	default:
		// Requests this server doesn't know get an empty answer, so the
		// editor doesn't wait; notifications are ignored
		if m.ID != nil {
			s.reply(m.ID, nil)
		}
	}
}

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// update keeps the new text of a file and sends its problems.
func (s *Server) update(uri, text string) {
	s.files[uri] = text
	dir := ""
	if path := uriPath(uri); path != "" {
		dir = filepath.Dir(path)
	}
	diagnostics := []any{}
	for _, d := range s.check(text, dir) {
		line, column := max(d.Line-1, 0), max(d.Column-1, 0)
		diagnostics = append(diagnostics, map[string]any{
			"range": map[string]any{
				"start": position{line, column},
				"end":   position{line, column + 1},
			},
			"severity": 1,
			"source":   "targascript",
			"message":  d.Message,
		})
	}
	s.notify("textDocument/publishDiagnostics", map[string]any{"uri": uri, "diagnostics": diagnostics})
}

// uriPath returns the path of a file:// URI, or "".
func uriPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return ""
	}
	return u.Path
}

var keywords = []string{"let", "const", "fn", "if", "else", "return", "repeat", "in", "break",
	"continue", "load", "as", "await", "try", "catch", "finally", "throw", "match", "true", "false", "null", "this"}

// completions lists the names the file declares, the builtins and the
// keywords.
func (s *Server) completions(uri string) []any {
	items := []any{}
	seen := map[string]bool{}
	for _, d := range declarations(s.files[uri]) {
		if !seen[d.name] {
			seen[d.name] = true
			kind := 6 // variable
			if d.kind == 12 {
				kind = 3 // function
			}
			items = append(items, map[string]any{"label": d.name, "kind": kind})
		}
	}
	for _, name := range evaluator.BuiltinNames() {
		item := map[string]any{"label": name, "kind": 3}
		if doc, ok := builtinDocs[name]; ok {
			item["detail"] = doc[0]
			item["documentation"] = doc[1]
		}
		items = append(items, item)
	}
	for _, keyword := range keywords {
		items = append(items, map[string]any{"label": keyword, "kind": 14})
	}
	return items
}

// wordAt returns the name under a position, and whether a dot comes before
// it, as in a property.
func (s *Server) wordAt(uri string, at position) (string, bool) {
	lines := strings.Split(s.files[uri], "\n")
	if at.Line >= len(lines) {
		return "", false
	}
	line := lines[at.Line]
	start, end := min(at.Character, len(line)), min(at.Character, len(line))
	for start > 0 && isNameByte(line[start-1]) {
		start--
	}
	for end < len(line) && isNameByte(line[end]) {
		end++
	}
	return line[start:end], start > 0 && line[start-1] == '.'
}

// definition returns where the name under a position is declared in the
// file.
func (s *Server) definition(uri string, at position) any {
	name, property := s.wordAt(uri, at)
	if name == "" || property {
		return nil
	}
	d, ok := definition(s.files[uri], name, at.Line, at.Character)
	if !ok {
		return nil
	}
	return map[string]any{"uri": uri, "range": nameRange(d)}
}

// format returns the edit that lays the file out like targascript fmt: the
// whole text replaced, or no edits when it's laid out already.
func (s *Server) format(uri string) []any {
	text := s.files[uri]
	formatted := format.Source(text)
	if formatted == text {
		return []any{}
	}
	lines := strings.Split(text, "\n")
	end := position{len(lines) - 1, len(lines[len(lines)-1])}
	return []any{map[string]any{
		"range":   map[string]any{"start": position{0, 0}, "end": end},
		"newText": formatted,
	}}
}

// symbols lists the functions, variables and constants declared at the top
// level of the file, for an outline.
func (s *Server) symbols(uri string) []any {
	symbols := []any{}
	for _, d := range declarations(s.files[uri]) {
		if !d.topLevel() {
			continue
		}
		symbols = append(symbols, map[string]any{
			"name":           d.name,
			"kind":           d.kind,
			"range":          nameRange(d),
			"selectionRange": nameRange(d),
		})
	}
	return symbols
}

func nameRange(d declaration) map[string]any {
	return map[string]any{
		"start": position{d.line, d.column},
		"end":   position{d.line, d.column + len(d.name)},
	}
}

// hover shows the signature and description of the builtin under the
// cursor.
func (s *Server) hover(uri string, at position) any {
	name, property := s.wordAt(uri, at)
	doc, ok := builtinDocs[name]
	if !ok || property {
		return nil
	}
	return map[string]any{
		"contents": map[string]any{
			"kind":  "markdown",
			"value": "```targascript\n" + doc[0] + "\n```\n" + doc[1],
		},
	}
}

func isNameByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}
