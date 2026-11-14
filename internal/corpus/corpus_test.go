package corpus

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/ragfuzz/internal/config"
)

func TestExtractEntities(t *testing.T) {
	text := `In API v2.4, only users with the admin role can update the rate_limit_rps configuration to 500req/s.
Deprecated in v1.0, viewers cannot modify auth_token.`

	entities := ExtractEntities(text)
	if len(entities) == 0 {
		t.Fatal("expected extracted entities, got 0")
	}

	foundVersion := false
	foundRole := false
	foundParam := false
	foundNumber := false

	for _, e := range entities {
		if e.Type == EntityVersion && (e.Value == "v2.4" || e.Value == "v1.0") {
			foundVersion = true
		}
		if e.Type == EntityRole && (e.Value == "admin" || e.Value == "viewer") {
			foundRole = true
		}
		if e.Type == EntityParam && (e.Value == "rate_limit_rps" || e.Value == "auth_token") {
			foundParam = true
		}
		if e.Type == EntityNumber && (e.Value == "500req/s" || e.Value == "500") {
			foundNumber = true
		}
	}

	if !foundVersion {
		t.Error("expected version entity (v2.4 or v1.0)")
	}
	if !foundRole {
		t.Error("expected role entity (admin or viewer)")
	}
	if !foundParam {
		t.Error("expected param entity (rate_limit_rps or auth_token)")
	}
	if !foundNumber {
		t.Error("expected number entity (500req/s)")
	}
}

func TestIngestCorpusMarkdown(t *testing.T) {
	mdContent := `# Enterprise SSO Configuration
Single Sign-On allows enterprise organizations to authenticate users via SAML 2.0.

## Supported Identity Providers
We support Okta, Azure AD, and Google Workspace. PingFederate is not supported in the Starter tier.

## Token Expiration Policy
Authentication tokens issued via SSO expire strictly after 3600 seconds (1 hour). Refresh tokens require MFA.`

	tmpDir := t.TempDir()
	docPath := filepath.Join(tmpDir, "sso.md")
	if err := os.WriteFile(docPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("failed to write test markdown: %v", err)
	}

	cfg := &config.CorpusConfig{
		Paths:          []string{docPath},
		ChunkSize:      500,
		ChunkOverlap:   50,
		MinChunkLength: 20,
	}

	chunks, err := IngestCorpus(cfg)
	if err != nil {
		t.Fatalf("IngestCorpus failed: %v", err)
	}

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// Verify headings and entities
	hasHeading := false
	for _, ch := range chunks {
		if ch.Heading == "Supported Identity Providers" || ch.Heading == "Token Expiration Policy" {
			hasHeading = true
		}
	}
	if !hasHeading {
		t.Error("expected preserved markdown section headings")
	}

	// Build topology
	topology := BuildTopology(chunks)
	if topology == nil {
		t.Fatal("BuildTopology returned nil")
	}

	// Test BM25 Search
	results := topology.SearchBM25("Okta Azure Google Workspace", 1)
	if len(results) == 0 {
		t.Fatal("expected search results from BM25")
	}
	if results[0].Heading != "Supported Identity Providers" {
		t.Errorf("expected top result to be 'Supported Identity Providers', got %q", results[0].Heading)
	}
}
