package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSnapshot(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"app.tg":            "print(1)",
		"notes.txt":         "",
		"lib/util.tg":       "",
		".git/hooks.tg":     "",
		"node_modules/x.tg": "",
	})
	first, err := snapshot(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Only .tg files outside hidden directories and node_modules count
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("changed"), 0o644)
	os.WriteFile(filepath.Join(dir, ".git/hooks.tg"), []byte("changed"), 0o644)
	os.WriteFile(filepath.Join(dir, "node_modules/x.tg"), []byte("changed"), 0o644)
	if same, _ := snapshot(dir); same != first {
		t.Errorf("snapshot changed for files it should ignore")
	}

	os.WriteFile(filepath.Join(dir, "lib/util.tg"), []byte("let x = 1"), 0o644)
	if changed, _ := snapshot(dir); changed == first {
		t.Errorf("snapshot didn't change after editing lib/util.tg")
	}
}

func TestWithoutWatch(t *testing.T) {
	got := withoutWatch([]string{"test", "--watch", "-run", "x"})
	if want := []string{"test", "-run", "x"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
