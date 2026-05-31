package main

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestValidateWebUIRequiresIndex(t *testing.T) {
	err := validateWebUI(fstest.MapFS{
		"placeholder.txt": {Data: []byte("not a release build")},
	})
	if err == nil {
		t.Fatal("expected missing index error")
	}
	if !strings.Contains(err.Error(), "missing index.html") {
		t.Fatalf("expected missing index hint, got %v", err)
	}
}

func TestValidateWebUIAcceptsIndex(t *testing.T) {
	err := validateWebUI(fstest.MapFS{
		"index.html": {Data: []byte("<html>Memoir</html>")},
	})
	if err != nil {
		t.Fatalf("expected valid web UI, got %v", err)
	}
}
