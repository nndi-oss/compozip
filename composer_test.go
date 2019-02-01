package main

import (
	"net/http"
	"testing"
)

type httpResponseWriter struct {
}

func (w httpResponseWriter) Header() http.Header {
	return nil
}

func (w httpResponseWriter) Write([]byte) (int, error) {
	return -1, nil
}

func (w httpResponseWriter) WriteHeader(int) {
	return
}

func TestParseComposerJSON(t *testing.T) {
	composerProject, err := parseComposerJSON(
		httpResponseWriter{},
		[]byte(dummyComposer),
		"composer.json",
	)

	if err != nil {
		t.Errorf("Expected err to be nil but got %v", err)
	}
	if composerProject == nil {
		t.Error("Expected composerProject but got nil")
	}
	if composerProject.ProjectName != "compozip/generated" {
		t.Errorf("composerProject's ProjectName is wrong. Got '%s' Expected '%s'", composerProject.ProjectName, "compozip/generated")
	}
	if composerProject.isComposerLock {
		t.Error("Expected isComposerLock to be 'false'")
	}
}
