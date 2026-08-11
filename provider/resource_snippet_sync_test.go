package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestSnippetResourceSchemaSyncNodesOnChange(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewSnippetResource().Schema(t.Context(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	attr, ok := resp.Schema.Attributes["sync_nodes_on_change"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("sync_nodes_on_change attribute type = %T, want schema.BoolAttribute", resp.Schema.Attributes["sync_nodes_on_change"])
	}
	if !attr.Optional || attr.Required || attr.Computed {
		t.Fatalf("sync_nodes_on_change must be optional-only: %#v", attr)
	}
}

func TestSnippetSyncVersionGateRunsBeforeMutation(t *testing.T) {
	t.Parallel()

	var mutationCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/system/metadata" {
			_, _ = io.WriteString(w, `{"response":{"version":"3.2.2"}}`)
			return
		}
		mutationCalls++
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	r := &snippetResource{client: client}
	if err := r.requireSyncSupport(context.Background()); err == nil {
		t.Fatal("requireSyncSupport() error = nil, want unsupported-version error")
	}
	if mutationCalls != 0 {
		t.Fatalf("mutation calls = %d, want 0", mutationCalls)
	}
}

func TestDeleteSnippetForSyncAcceptsAlreadyDeletedSnippet(t *testing.T) {
	t.Parallel()

	var deleteCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/api/snippets" {
			deleteCalls++
			w.WriteHeader(http.StatusNotFound)
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	r := &snippetResource{client: client}
	if err := r.deleteSnippetForSync(context.Background(), "already-deleted"); err != nil {
		t.Fatalf("deleteSnippetForSync() error = %v, want nil", err)
	}
	if deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", deleteCalls)
	}
}
