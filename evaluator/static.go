package evaluator

import (
	"path"
	"strings"

	"github.com/migueltarga/TargaScript/object"
)

func init() {
	builtins["staticFile"] = &object.Builtin{Name: "staticFile", Fn: staticFileBuiltin}
}

// contentType is the type of the files a site usually has, by extension.
func contentType(k string) (string, bool) {
	switch k {
	case ".html":
		return "text/html; charset=utf-8", true
	case ".css":
		return "text/css; charset=utf-8", true
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8", true
	case ".json":
		return "application/json", true
	case ".txt", ".tg":
		return "text/plain; charset=utf-8", true
	case ".md":
		return "text/markdown; charset=utf-8", true
	case ".svg":
		return "image/svg+xml", true
	case ".png":
		return "image/png", true
	case ".jpg", ".jpeg":
		return "image/jpeg", true
	case ".gif":
		return "image/gif", true
	case ".webp":
		return "image/webp", true
	case ".ico":
		return "image/x-icon", true
	case ".wasm":
		return "application/wasm", true
	case ".pdf":
		return "application/pdf", true
	case ".woff":
		return "font/woff", true
	case ".woff2":
		return "font/woff2", true
	}
	return "", false
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
	contentType, ok := contentType(strings.ToLower(path.Ext(file)))
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
