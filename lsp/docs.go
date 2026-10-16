package lsp

import (
	_ "embed"
	"encoding/json"
)

// builtins.json has the signature and description of each builtin. The VS
// Code extension uses it too; its build copies it.
//
//go:embed builtins.json
var builtinsJSON []byte

var builtinDocs = func() map[string][2]string {
	docs := map[string][2]string{}
	json.Unmarshal(builtinsJSON, &docs)
	return docs
}()
