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

func linkTestClient(srv *httptest.Server) *Client {
	return &Client{
		baseURL:      srv.URL,
		authHeader:   "Bearer test-token",
		httpClient:   srv.Client(),
		instanceType: InstanceServer,
	}
}

// Proves the subject of the sentence goes into inwardIssue, which is how Jira
// Server actually renders it (verified against jira.mts.ru, 2026-09-29):
// "PROJ-1 parent of PROJ-2" is sent as inward=PROJ-1, outward=PROJ-2. Sending it
// the intuitive way round silently produces the opposite relationship, so this
// is the claim worth pinning.
func TestCreateIssueLink_SubjectIsSentAsInwardIssue(t *testing.T) {
	var got createIssueLinkRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/issueLink" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	if err := linkTestClient(srv).CreateIssueLink("Parent", "PROJ-1", "PROJ-2"); err != nil {
		t.Fatalf("CreateIssueLink: %v", err)
	}
	if got.Type.Name != "Parent" {
		t.Fatalf("type = %q", got.Type.Name)
	}
	if got.InwardIssue.Key != "PROJ-1" {
		t.Fatalf("inwardIssue = %q, want the subject PROJ-1", got.InwardIssue.Key)
	}
	if got.OutwardIssue.Key != "PROJ-2" {
		t.Fatalf("outwardIssue = %q, want the object PROJ-2", got.OutwardIssue.Key)
	}
}

// Negative: the malformed calls must be refused locally, with nothing sent, so a
// batch cannot half-apply and a typo cannot create a self-link.
func TestCreateIssueLink_RefusesMalformedCallsWithoutCallingJira(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("Jira must not be called, got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()
	c := linkTestClient(srv)

	cases := []struct {
		name            string
		typeName, a, b  string
		wantErrContains string
	}{
		{"no type", "", "PROJ-1", "PROJ-2", "link type is required"},
		{"no subject", "Parent", "", "PROJ-2", "both issue keys are required"},
		{"no object", "Parent", "PROJ-1", "", "both issue keys are required"},
		{"self link", "Parent", "PROJ-1", "PROJ-1", "itself"},
		{"self link, different case and spacing", "Parent", "proj-1", " PROJ-1 ", "itself"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := c.CreateIssueLink(tc.typeName, tc.a, tc.b)
			if err == nil {
				t.Fatalf("expected a refusal")
			}
			if !strings.Contains(err.Error(), tc.wantErrContains) {
				t.Fatalf("error = %q, want it to mention %q", err, tc.wantErrContains)
			}
		})
	}
}

// Positive control for the refusals above: a well-formed call on the same client
// does reach Jira, so the test above is not passing because everything fails.
func TestCreateIssueLink_WellFormedCallReachesJira(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	if err := linkTestClient(srv).CreateIssueLink("Parent", "PROJ-1", "PROJ-2"); err != nil {
		t.Fatalf("CreateIssueLink: %v", err)
	}
	if !called {
		t.Fatalf("Jira was not called")
	}
}

// Proves an unknown type is refused by name before any link is created, and that
// the refusal tells the caller what the instance does offer.
func TestResolveIssueLinkType_MatchesCaseInsensitivelyAndNamesAlternatives(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/issueLinkType" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"issueLinkTypes":[{"id":"10","name":"Parent","inward":"child of","outward":"parent of"},{"id":"11","name":"Relates","inward":"relates to","outward":"relates to"}]}`)
	}))
	defer srv.Close()
	c := linkTestClient(srv)

	got, err := c.ResolveIssueLinkType("parent")
	if err != nil {
		t.Fatalf("ResolveIssueLinkType: %v", err)
	}
	if got.Name != "Parent" || got.Outward != "parent of" || got.Inward != "child of" {
		t.Fatalf("resolved = %#v", got)
	}

	_, err = c.ResolveIssueLinkType("Subtask")
	if err == nil {
		t.Fatalf("expected a refusal for a type the instance does not have")
	}
	if !strings.Contains(err.Error(), "Parent") || !strings.Contains(err.Error(), "Relates") {
		t.Fatalf("error = %q, want it to list the available types", err)
	}
}

// Proves a link is read back from the side it sits on: the issue in the response
// is the OTHER end, and which field carries it says which wording applies.
func TestGetIssueLinks_ReadsBothSides(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("fields"); got != "issuelinks" {
			t.Fatalf("fields = %q, want only issuelinks", got)
		}
		fmt.Fprint(w, `{"fields":{"issuelinks":[
			{"id":"1","type":{"name":"Parent","inward":"child of","outward":"parent of"},"outwardIssue":{"key":"PROJ-2","fields":{"summary":"child"}}},
			{"id":"2","type":{"name":"Blocks","inward":"is blocked by","outward":"blocks"},"inwardIssue":{"key":"PROJ-9","fields":{"summary":"blocker"}}}
		]}}`)
	}))
	defer srv.Close()

	links, err := linkTestClient(srv).GetIssueLinks("PROJ-1")
	if err != nil {
		t.Fatalf("GetIssueLinks: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("got %d links", len(links))
	}
	if links[0].OutwardIssue == nil || links[0].OutwardIssue.Key != "PROJ-2" || links[0].InwardIssue != nil {
		t.Fatalf("link 0 = %#v", links[0])
	}
	if links[1].InwardIssue == nil || links[1].InwardIssue.Key != "PROJ-9" || links[1].OutwardIssue != nil {
		t.Fatalf("link 1 = %#v", links[1])
	}
}

// Negative: an empty id must not turn into a DELETE against the collection.
func TestDeleteIssueLink_RefusesEmptyID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("Jira must not be called, got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	if err := linkTestClient(srv).DeleteIssueLink(""); err == nil {
		t.Fatalf("expected a refusal")
	}
}
