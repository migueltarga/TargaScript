package compiler

import "testing"

func TestResolveLoadPathWithURLs(t *testing.T) {
	tests := []struct{ dir, file, want string }{
		{"/home/ana", "https://example.com/lib/a.tg", "https://example.com/lib/a.tg"},
		{"https://example.com/lib", "b.tg", "https://example.com/lib/b.tg"},
		{"https://example.com/lib", "./sub/c.tg", "https://example.com/lib/sub/c.tg"},
		{"https://example.com/lib/deep", "../d.tg", "https://example.com/lib/d.tg"},
		{"https://example.com/lib", "../../../e.tg", "https://example.com/e.tg"},
		{"https://example.com/lib", "/root.tg", "https://example.com/root.tg"},
		{"http://localhost:8080", "f.tg", "http://localhost:8080/f.tg"},
	}
	for _, tt := range tests {
		got, err := ResolveLoadPath(tt.dir, tt.file)
		if err != nil || got != tt.want {
			t.Errorf("ResolveLoadPath(%q, %q) = %q, %v; want %q", tt.dir, tt.file, got, err, tt.want)
		}
	}
}

func TestModuleDir(t *testing.T) {
	tests := []struct{ file, want string }{
		{"https://example.com/lib/a.tg", "https://example.com/lib"},
		{"https://example.com/a.tg", "https://example.com"},
		{"https://example.com", "https://example.com"},
		{"/home/ana/a.tg", "/home/ana"},
	}
	for _, tt := range tests {
		if got := ModuleDir(tt.file); got != tt.want {
			t.Errorf("ModuleDir(%q) = %q, want %q", tt.file, got, tt.want)
		}
	}
}
