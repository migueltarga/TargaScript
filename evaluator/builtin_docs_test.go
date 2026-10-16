package evaluator

import (
	"encoding/json"
	"os"
	"testing"
)

// The VS Code extension shows the docs in builtins.json when the mouse is
// over a builtin, so every builtin needs an entry, and only builtins.
func TestEveryBuiltinHasDocs(t *testing.T) {
	data, err := os.ReadFile("../targascript-vscode/builtins.json")
	if err != nil {
		t.Fatal(err)
	}
	docs := map[string][2]string{}
	if err := json.Unmarshal(data, &docs); err != nil {
		t.Fatal(err)
	}
	for _, name := range BuiltinNames() {
		doc, ok := docs[name]
		if !ok || doc[0] == "" || doc[1] == "" {
			t.Errorf("builtins.json has no signature and description for %s", name)
		}
		delete(docs, name)
	}
	for name := range docs {
		t.Errorf("builtins.json describes %s, which isn't a builtin", name)
	}
}
