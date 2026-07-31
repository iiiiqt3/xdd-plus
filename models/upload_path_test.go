package models

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUploadAbsPath(t *testing.T) {
	root := t.TempDir()
	oldExecPath := ExecPath
	ExecPath = root
	t.Cleanup(func() { ExecPath = oldExecPath })

	uploadsRoot := filepath.Join(root, "uploads")
	guideDir := filepath.Join(uploadsRoot, "guide", "common")
	if err := os.MkdirAll(guideDir, 0755); err != nil {
		t.Fatal(err)
	}
	sample := filepath.Join(guideDir, "demo.png")
	if err := os.WriteFile(sample, []byte("png"), 0644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		input   string
		wantRel string
		ok      bool
	}{
		{name: "splat style", input: "guide/common/demo.png", wantRel: sample, ok: true},
		{name: "leading slash", input: "/guide/common/demo.png", wantRel: sample, ok: true},
		{name: "uploads prefix", input: "/uploads/guide/common/demo.png", wantRel: sample, ok: true},
		{name: "traversal blocked", input: "../etc/passwd", ok: false},
		{name: "empty blocked", input: "", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ResolveUploadAbsPath(tc.input)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v (input=%q)", ok, tc.ok, tc.input)
			}
			if !tc.ok {
				return
			}
			if got != tc.wantRel {
				t.Fatalf("got %q, want %q", got, tc.wantRel)
			}
		})
	}
}
