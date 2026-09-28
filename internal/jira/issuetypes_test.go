package jira

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The shape a localized instance actually has: English names absent, several
// subtask types, and two of them sharing one name (jira.mts.ru, 2026-09-29).
const localizedIssueTypes = `[
  {"id":"10107","name":"Подзадача на разработку","subtask":true},
  {"id":"10109","name":"Подзадача на тестирование","subtask":true},
  {"id":"10200","name":"Подзадача","subtask":true},
  {"id":"10401","name":"Подзадача","subtask":true},
  {"id":"10001","name":"История","subtask":false}
]`

func issueTypeServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/issuetype" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, localizedIssueTypes)
	}))
}

// Proves a type can be named the way the instance names it, or by id. The CLI's
// English shorthand does not exist on a localized Jira, so resolving against the
// instance is the only way to create anything there at all.
func TestResolveIssueType_ByLocalizedNameAndByID(t *testing.T) {
	srv := issueTypeServer(t)
	defer srv.Close()
	c := fieldTestClient(srv)

	byName, err := c.ResolveIssueType("Подзадача на разработку")
	if err != nil || byName.ID != "10107" {
		t.Fatalf("by name: %#v %v", byName, err)
	}
	byID, err := c.ResolveIssueType("10109")
	if err != nil || byID.Name != "Подзадача на тестирование" {
		t.Fatalf("by id: %#v %v", byID, err)
	}
}

// Negative: two types share the name "Подзадача" on this instance. Picking
// either would create the issue under the wrong type, which looks like success.
func TestResolveIssueType_RefusesAmbiguousAndUnknown(t *testing.T) {
	srv := issueTypeServer(t)
	defer srv.Close()
	c := fieldTestClient(srv)

	_, err := c.ResolveIssueType("Подзадача")
	if err == nil {
		t.Fatal("expected a refusal for the duplicated name")
	}
	for _, want := range []string{"ambiguous", "10200", "10401"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q should mention %q", err, want)
		}
	}

	if _, err := c.ResolveIssueType("Sub-task"); err == nil {
		t.Fatal("expected a refusal: this instance has no English type names")
	}
}
