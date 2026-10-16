package evaluator

import (
	"path"
	"strings"

	"github.com/migueltarga/TargaScript/object"
)

func init() {
	builtins["staticFile"] = &object.Builtin{Name: "staticFile", Fn: staticFileBuiltin}
}

// contentTypes are the types of the files a site usually has, by extension.
var contentTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".mjs":   "text/javascript; charset=utf-8",
	".json":  "application/json",
	".txt":   "text/plain; charset=utf-8",
	".md":    "text/markdown; charset=utf-8",
	".tg":    "text/plain; charset=utf-8",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".webp":  "image/webp",
	".ico":   "image/x-icon",
	".wasm":  "application/wasm",
	".pdf":   "application/pdf",
	".woff":  "font/woff",
	".woff2": "font/woff2",
}

// staticFileBuiltin is staticFile(dir, urlPath): a response with the file
// at urlPath inside dir, index.html for a directory (after redirecting to
// the path with a slash), or a 404. The path
// can't leave dir, and hidden files, whose names start with a dot, aren't
// served.
func staticFileBuiltin(args ...object.Object) object.Object {
	values, err := stringArgs("staticFile", args, 2)
	if err != nil {
		return err
	}
	if system == nil {
		return noSystem("staticFile")
	}
	dir, urlPath := values[0], values[1]

	clean := path.Clean("/" + urlPath)
	for _, part := range strings.Split(clean, "/") {
		if strings.HasPrefix(part, ".") {
			return notFound()
		}
	}
	file := path.Join(dir, clean)
	if _, listErr := system.ListDir(file); listErr == nil {
		// Relative links in the index need the URL to end in a slash
		if !strings.HasSuffix(urlPath, "/") {
			return &Response{Status: 301, Headers: map[string]string{"Location": clean + "/"}, Body: &object.String{Value: ""}}
		}
		file = path.Join(file, "index.html")
	}

	content, readErr := system.ReadFile(file)
	if readErr != nil {
		return notFound()
	}
	contentType, ok := contentTypes[strings.ToLower(path.Ext(file))]
	if !ok {
		contentType = "application/octet-stream"
	}
	return &Response{Status: 200, Headers: map[string]string{"Content-Type": contentType}, Body: &object.String{Value: content}}
}

func notFound() *Response {
	return &Response{
		Status:  404,
		Headers: map[string]string{"Content-Type": "text/plain; charset=utf-8"},
		Body:    &object.String{Value: "Not found\n"},
	}
}
