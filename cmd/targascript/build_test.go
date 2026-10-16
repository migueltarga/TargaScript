package main

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"testing"
)

func TestBundleCollectsLoads(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"main.tg":      "load \"lib/a.tg\"\nfn f() {\n  load \"lib/b.tg\" as b\n}",
		"lib/a.tg":     "load \"./c.tg\"\nlet a = 1",
		"lib/b.tg":     "let b = 2",
		"lib/c.tg":     "load \"a.tg\"\nlet c = 3",
		"lib/other.tg": "never loaded",
	})
	main := filepath.Join(dir, "main.tg")
	b := bundle{Main: main, Files: map[string]string{}}
	if err := b.add(main); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.tg", "lib/a.tg", "lib/b.tg", "lib/c.tg"} {
		if _, ok := b.Files[filepath.Join(dir, name)]; !ok {
			t.Errorf("%s is missing from the bundle", name)
		}
	}
	if len(b.Files) != 4 {
		t.Errorf("bundle has %d files, want 4", len(b.Files))
	}

	missing := bundle{Files: map[string]string{}}
	if err := missing.add(filepath.Join(dir, "nope.tg")); err == nil {
		t.Error("a missing file should fail the build")
	}
}

func TestBundleSize(t *testing.T) {
	var executable bytes.Buffer
	executable.WriteString("the runtime")
	executable.WriteString(`{"main":"x"}`)
	binary.Write(&executable, binary.LittleEndian, uint64(len(`{"main":"x"}`)))
	executable.WriteString(bundleMagic)

	size, ok := bundleSize(executable.Bytes())
	if !ok || executable.Len()-size != len("the runtime") {
		t.Errorf("bundleSize = %d, %t", size, ok)
	}
	if _, ok := bundleSize([]byte("a plain executable")); ok {
		t.Error("found a bundle in a plain executable")
	}

	// The test binary has none, so it runs as itself
	if _, ok := readBundle(); ok {
		t.Error("the test binary has a bundle")
	}
}
