package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// newContractClientAtVersion builds a client whose version detection is
// pre-seeded, so version-branched routes can be exercised without a real
// /api/system/metadata round trip.
func newContractClientAtVersion(t *testing.T, handler http.HandlerFunc, version string) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(ClientConfig{Endpoint: server.URL, APIToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	client.serverVersion = version
	return client
}

func sharedListHandler(t *testing.T, method string, check func(*http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			t.Errorf("method = %s, want %s", r.Method, method)
		}
		check(r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{"name":"list","config":{}}}`)
	}
}

// TestClientV3_4SharedListRoutes pins the Remnawave 3.4 shared-list
// identifier contract: GET moved the name into a query parameter and DELETE
// moved it into a JSON body, because names may now contain "/".
func TestClientV3_4SharedListRoutes(t *testing.T) {
	t.Parallel()

	t.Run("3.4 GET encodes the name as a query parameter", func(t *testing.T) {
		client := newContractClientAtVersion(t, sharedListHandler(t, http.MethodGet, func(r *http.Request) {
			if r.URL.Path != "/api/node-plugins/shared-lists/by-name" {
				t.Errorf("path = %s", r.URL.Path)
			}
			if got := r.URL.Query().Get("name"); got != "a/b" {
				t.Errorf("query name = %q", got)
			}
		}), "3.4")
		list, err := client.GetSharedListByName(context.Background(), "a/b")
		if err != nil {
			t.Fatal(err)
		}
		if list.Name != "list" {
			t.Errorf("list name = %q", list.Name)
		}
	})

	t.Run("3.4 DELETE sends the name in the body", func(t *testing.T) {
		client := newContractClientAtVersion(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.Path != "/api/node-plugins/shared-lists" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			want := map[string]any{"name": "a/b"}
			if got := decodeRequestObject(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
			w.WriteHeader(http.StatusNoContent)
		}, "3.4")
		if err := client.DeleteSharedList(context.Background(), "a/b"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("3.3 keeps the path-parameter routes", func(t *testing.T) {
		client := newContractClientAtVersion(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/node-plugins/shared-lists/priv" {
				t.Errorf("path = %s", r.URL.Path)
			}
			if r.Body != nil && r.ContentLength > 0 {
				t.Errorf("unexpected body on %s", r.Method)
			}
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"response":{"name":"list","config":{}}}`)
		}, "3.3")
		if _, err := client.GetSharedListByName(context.Background(), "priv"); err != nil {
			t.Fatal(err)
		}
		if err := client.DeleteSharedList(context.Background(), "priv"); err != nil {
			t.Fatal(err)
		}
	})
}

// TestClientV3_4HostSquadsAdaptation pins the host squad translation:
// 3.4 panels receive internalSquads {mode, squads} and never the legacy
// excludedInternalSquads array; pre-3.4 panels keep the legacy array and
// reject configurations that set the new block.
func TestClientV3_4HostSquadsAdaptation(t *testing.T) {
	t.Parallel()

	hostResponse := `{"response":{"uuid":"h1","remark":"r","address":"a","port":1}}`
	excluded := []string{"s1"}

	run := func(t *testing.T, version string, host *Host, verify func(t *testing.T, body map[string]any)) {
		t.Helper()
		for _, call := range []struct {
			name   string
			invoke func(*Client) error
			method string
		}{
			{"create", func(c *Client) error { _, err := c.CreateHost(context.Background(), host); return err }, http.MethodPost},
			{"update", func(c *Client) error { _, err := c.UpdateHost(context.Background(), host); return err }, http.MethodPatch},
		} {
			t.Run(call.name, func(t *testing.T) {
				client := newContractClientAtVersion(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Method != call.method || r.URL.Path != "/api/hosts" {
						t.Errorf("request = %s %s", r.Method, r.URL.Path)
					}
					verify(t, decodeRequestObject(t, r))
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, hostResponse)
				}, version)
				if err := call.invoke(client); err != nil {
					t.Fatal(err)
				}
			})
		}
	}

	t.Run("3.4 translates excludedInternalSquads to EXCLUDE mode", func(t *testing.T) {
		run(t, "3.4", &Host{Remark: "r", Address: "a", Port: 1, ExcludedInternalSquads: &excluded},
			func(t *testing.T, body map[string]any) {
				want := map[string]any{"mode": "EXCLUDE", "squads": []any{"s1"}}
				if got := body["internalSquads"]; !reflect.DeepEqual(got, want) {
					t.Errorf("internalSquads = %#v, want %#v", got, want)
				}
				if _, ok := body["excludedInternalSquads"]; ok {
					t.Error("excludedInternalSquads sent to a 3.4 panel")
				}
			})
	})

	t.Run("3.4 passes internalSquads through without the legacy key", func(t *testing.T) {
		run(t, "3.4", &Host{Remark: "r", Address: "a", Port: 1, InternalSquads: &HostInternalSquads{Mode: "ALLOW_ONLY", Squads: []string{"a", "b"}}},
			func(t *testing.T, body map[string]any) {
				want := map[string]any{"mode": "ALLOW_ONLY", "squads": []any{"a", "b"}}
				if got := body["internalSquads"]; !reflect.DeepEqual(got, want) {
					t.Errorf("internalSquads = %#v, want %#v", got, want)
				}
				if _, ok := body["excludedInternalSquads"]; ok {
					t.Error("excludedInternalSquads sent to a 3.4 panel")
				}
			})
	})

	t.Run("3.4 omits both keys when neither is configured", func(t *testing.T) {
		run(t, "3.4", &Host{Remark: "r", Address: "a", Port: 1},
			func(t *testing.T, body map[string]any) {
				if _, ok := body["internalSquads"]; ok {
					t.Error("internalSquads sent although not configured")
				}
				if _, ok := body["excludedInternalSquads"]; ok {
					t.Error("excludedInternalSquads sent although not configured")
				}
			})
	})

	t.Run("3.3 keeps the legacy array", func(t *testing.T) {
		run(t, "3.3", &Host{Remark: "r", Address: "a", Port: 1, ExcludedInternalSquads: &excluded},
			func(t *testing.T, body map[string]any) {
				if got := body["excludedInternalSquads"]; !reflect.DeepEqual(got, []any{"s1"}) {
					t.Errorf("excludedInternalSquads = %#v", got)
				}
				if _, ok := body["internalSquads"]; ok {
					t.Error("internalSquads sent to a pre-3.4 panel")
				}
			})
	})

	t.Run("pre-3.4 rejects the internal_squads block", func(t *testing.T) {
		client := &Client{serverVersion: "3.3"}
		_, err := client.CreateHost(context.Background(), &Host{Remark: "r", Address: "a", Port: 1,
			InternalSquads: &HostInternalSquads{Mode: "EXCLUDE", Squads: []string{}}})
		if err == nil {
			t.Fatal("CreateHost accepted internalSquads on a pre-3.4 panel")
		}
	})
}
