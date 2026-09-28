package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Regression witness: Server/DC v2 refuses an ADF description outright
// ("Значение операции должно быть строкой"), so CreateIssueFields must be able
// to carry a plain string. It was typed *ADFDoc, which made creating any issue
// with a description impossible on a Server instance.
func TestCreateIssueFields_CarryAPlainStringDescription(t *testing.T) {
	var sent map[string]map[string]json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &sent); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"1","key":"PROJ-1"}`))
	}))
	defer srv.Close()

	req := &CreateIssueRequest{Fields: CreateIssueFields{
		Project:     ProjectRef{Key: "PROJ"},
		IssueType:   IssueTypeRef{ID: "10107"},
		Summary:     "s",
		Description: "plain text",
	}}
	if _, err := fieldTestClient(srv).CreateIssue(req); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}

	got := string(sent["fields"]["description"])
	if got != `"plain text"` {
		t.Fatalf("description = %s, want a JSON string", got)
	}
	if id := string(sent["fields"]["issuetype"]); id != `{"id":"10107"}` {
		t.Fatalf("issuetype = %s, want it addressed by id", id)
	}
}

// The Cloud shape still works: an ADF document marshals as an object, so the
// fix did not trade one instance type for the other.
func TestCreateIssueFields_StillCarryADFForCloud(t *testing.T) {
	var sent map[string]map[string]json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &sent)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"1","key":"PROJ-1"}`))
	}))
	defer srv.Close()

	req := &CreateIssueRequest{Fields: CreateIssueFields{
		Project:     ProjectRef{Key: "PROJ"},
		IssueType:   IssueTypeRef{Name: "Task"},
		Summary:     "s",
		Description: NewADFText("rich text"),
	}}
	if _, err := fieldTestClient(srv).CreateIssue(req); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(sent["fields"]["description"], &doc); err != nil {
		t.Fatalf("description is not an object: %v", err)
	}
	if doc["type"] != "doc" {
		t.Fatalf("description = %v, want an ADF doc", doc)
	}
}
