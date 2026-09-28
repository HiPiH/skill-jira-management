package jira

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const fieldCatalogue = `[
  {"id":"summary","name":"Summary","custom":false},
  {"id":"customfield_13700","name":"Smart Checklist","custom":true},
  {"id":"customfield_13701","name":"Checklists","custom":true},
  {"id":"customfield_20001","name":"Team","custom":true},
  {"id":"customfield_20002","name":"Team","custom":true}
]`

func fieldTestClient(srv *httptest.Server) *Client {
	return &Client{baseURL: srv.URL, authHeader: "Bearer t", httpClient: srv.Client(), instanceType: InstanceServer}
}

func catalogueServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/field" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, fieldCatalogue)
	}))
}

// Proves a custom field can be addressed by its human name as well as its id —
// the id is what the API needs and nobody remembers customfield_13700.
func TestResolveFieldID_ByIDAndByName(t *testing.T) {
	srv := catalogueServer(t)
	defer srv.Close()
	c := fieldTestClient(srv)

	byID, err := c.ResolveFieldID("customfield_13700")
	if err != nil || byID.Name != "Smart Checklist" {
		t.Fatalf("by id: %#v %v", byID, err)
	}
	byName, err := c.ResolveFieldID("  smart checklist ")
	if err != nil || byName.ID != "customfield_13700" {
		t.Fatalf("by name: %#v %v", byName, err)
	}
}

// Negative: a name shared by two fields must be refused, not guessed. Picking
// one would write to the wrong custom field, which produces no error at all.
func TestResolveFieldID_RefusesAmbiguousAndUnknownNames(t *testing.T) {
	srv := catalogueServer(t)
	defer srv.Close()
	c := fieldTestClient(srv)

	_, err := c.ResolveFieldID("Team")
	if err == nil {
		t.Fatal("expected a refusal for the duplicated name")
	}
	for _, want := range []string{"ambiguous", "customfield_20001", "customfield_20002"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q should mention %q", err, want)
		}
	}

	if _, err := c.ResolveFieldID("Nonesuch"); err == nil {
		t.Fatal("expected a refusal for an unknown name")
	}
}

// Proves a cleared field is sent as JSON null, which is what Jira reads as
// "empty this", and that the write goes to the issue's own path.
func TestSetIssueFields_ClearsWithNull(t *testing.T) {
	var got map[string]map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/api/2/issue/PROJ-1" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := fieldTestClient(srv).SetIssueFields("PROJ-1", map[string]interface{}{"customfield_13700": nil}); err != nil {
		t.Fatalf("SetIssueFields: %v", err)
	}
	value, present := got["fields"]["customfield_13700"]
	if !present || value != nil {
		t.Fatalf("fields = %#v, want an explicit null", got["fields"])
	}
}

// Negative: nothing to write, or an empty field id, is refused locally.
func TestSetIssueFields_RefusesEmptyRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("Jira must not be called: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()
	c := fieldTestClient(srv)

	if err := c.SetIssueFields("PROJ-1", nil); err == nil {
		t.Fatal("expected a refusal for no fields")
	}
	if err := c.SetIssueFields("PROJ-1", map[string]interface{}{"  ": "x"}); err == nil {
		t.Fatal("expected a refusal for an empty field id")
	}
}

// Proves the raw passthrough sends the path unchanged, so a plugin endpoint
// outside /rest/api is reachable, and returns the body as received.
func TestRawRequest_PassesThePathThrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/com.example.plugin/1.0/thing" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	data, err := fieldTestClient(srv).RawRequest("GET", "/rest/com.example.plugin/1.0/thing", "")
	if err != nil || string(data) != `{"ok":true}` {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

// Proves a read asks for every field, which is the point: the typed model does
// not carry custom fields and a narrowed request would hide them.
func TestGetIssueRawFields_AsksForAllFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("fields"); got != "*all" {
			t.Fatalf("fields = %q", got)
		}
		fmt.Fprint(w, `{"fields":{"customfield_13700":["x"],"summary":"s"}}`)
	}))
	defer srv.Close()

	fields, err := fieldTestClient(srv).GetIssueRawFields("PROJ-1")
	if err != nil {
		t.Fatalf("GetIssueRawFields: %v", err)
	}
	if _, ok := fields["customfield_13700"]; !ok {
		t.Fatalf("custom field missing from %v", fields)
	}
}
