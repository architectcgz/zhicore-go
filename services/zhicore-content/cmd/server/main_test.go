package main

import (
	"strings"
	"testing"
)

func TestBuildModuleFailsFastUntilRuntimeDependenciesAreWired(t *testing.T) {
	err := buildModule()
	if err == nil {
		t.Fatal("buildModule() error = nil, want missing dependency error")
	}
	if !strings.Contains(err.Error(), "build content runtime module") || !strings.Contains(err.Error(), "Config") {
		t.Fatalf("buildModule() error = %v, want runtime Config dependency failure", err)
	}
}
