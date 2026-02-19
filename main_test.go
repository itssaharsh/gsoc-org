package main

import (
	"testing"
)

func TestOrgStruct(t *testing.T) {
	org := Org{Name: "Test Org", Year: 2025}
	if org.Name != "Test Org" {
		t.Errorf("Expected Name 'Test Org', got %s", org.Name)
	}
	if org.Year != 2025 {
		t.Errorf("Expected Year 2025, got %d", org.Year)
	}
}
