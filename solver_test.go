package main

import (
	"strconv"
	"testing"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestZoneNameForFQDN(t *testing.T) {
	cases := map[string]string{
		"_acme-challenge.example.com.":     "example.com",
		"_acme-challenge.www.example.com.": "example.com",
		"_acme-challenge.example.co.uk.":   "example.co.uk",
		"_acme-challenge.EXAMPLE.com":      "example.com",
	}
	for in, want := range cases {
		got, err := zoneNameForFQDN(in)
		if err != nil {
			t.Fatalf("zoneNameForFQDN(%q): %v", in, err)
		}
		if got != want {
			t.Errorf("zoneNameForFQDN(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRecordName(t *testing.T) {
	cases := map[string]string{
		"_acme-challenge.example.com.": "_acme-challenge.example.com.",
		"_acme-challenge.example.com":  "_acme-challenge.example.com.",
		"_ACME-Challenge.Example.com.": "_acme-challenge.example.com.",
	}
	for in, want := range cases {
		if got := recordName(in); got != want {
			t.Errorf("recordName(%q) = %q, want %q", in, got, want)
		}
	}
}

// The challenge key is stored as a TXT record in PowerDNS presentation format,
// i.e. wrapped in quotes. ACME keys are base64url so quoting only adds the
// surrounding quotes.
func TestChallengeContentQuoting(t *testing.T) {
	key := "abc123_-XYZ"
	if got := strconv.Quote(key); got != `"abc123_-XYZ"` {
		t.Errorf("strconv.Quote(%q) = %q, want %q", key, got, `"abc123_-XYZ"`)
	}
}

func TestLoadConfig(t *testing.T) {
	valid := &extapi.JSON{Raw: []byte(`{"projectId":"proj-abc","apiTokenSecretRef":{"name":"enum-token","key":"token"}}`)}
	cfg, err := loadConfig(valid)
	if err != nil {
		t.Fatalf("loadConfig valid: %v", err)
	}
	if cfg.ProjectID != "proj-abc" {
		t.Errorf("ProjectID = %q, want proj-abc", cfg.ProjectID)
	}
	if cfg.APITokenSecretRef.Name != "enum-token" || cfg.APITokenSecretRef.Key != "token" {
		t.Errorf("APITokenSecretRef = %+v", cfg.APITokenSecretRef)
	}

	if _, err := loadConfig(nil); err == nil {
		t.Error("loadConfig(nil) should error")
	}

	missingProject := &extapi.JSON{Raw: []byte(`{"apiTokenSecretRef":{"name":"x","key":"y"}}`)}
	if _, err := loadConfig(missingProject); err == nil {
		t.Error("loadConfig without projectId should error")
	}

	missingSecret := &extapi.JSON{Raw: []byte(`{"projectId":"proj-abc"}`)}
	if _, err := loadConfig(missingSecret); err == nil {
		t.Error("loadConfig without apiTokenSecretRef should error")
	}
}
