package jira

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// IssueLinkType is one of the link kinds the instance offers, with the wording
// each side of a link reads with.
type IssueLinkType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
}

type issueLinkTypesResponse struct {
	IssueLinkTypes []IssueLinkType `json:"issueLinkTypes"`
}

type issueLinkRef struct {
	Key string `json:"key"`
}

type issueLinkTypeRef struct {
	Name string `json:"name"`
}

type createIssueLinkRequest struct {
	Type         issueLinkTypeRef `json:"type"`
	InwardIssue  issueLinkRef     `json:"inwardIssue"`
	OutwardIssue issueLinkRef     `json:"outwardIssue"`
}

// ListIssueLinkTypes returns the link types configured on the instance.
func (c *Client) ListIssueLinkTypes() ([]IssueLinkType, error) {
	data, err := c.Get(c.apiPathFor("issueLinkType"), nil)
	if err != nil {
		return nil, fmt.Errorf("ListIssueLinkTypes: %w", err)
	}

	var resp issueLinkTypesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("ListIssueLinkTypes: failed to unmarshal: %w", err)
	}
	return resp.IssueLinkTypes, nil
}

// ResolveIssueLinkType finds a link type by name, case-insensitively. It refuses
// a name that is not configured rather than letting Jira fail on the create,
// and names what the instance does offer.
func (c *Client) ResolveIssueLinkType(name string) (IssueLinkType, error) {
	types, err := c.ListIssueLinkTypes()
	if err != nil {
		return IssueLinkType{}, err
	}

	for _, t := range types {
		if equalFoldTrimmed(t.Name, name) {
			return t, nil
		}
	}

	available := make([]string, 0, len(types))
	for _, t := range types {
		available = append(available, t.Name)
	}
	return IssueLinkType{}, fmt.Errorf("no link type %q on this instance; available: %v", name, available)
}

// CreateIssueLink links two issues. subjectKey is the issue that ends up reading
// with the type's outward wording and objectKey the one reading with the inward
// wording: with type "Parent" (outward "parent of", inward "child of"),
// subject=PROJ-1, object=PROJ-2 gives "PROJ-1 parent of PROJ-2".
//
// The request fields are the other way round from that reading. Verified against
// Jira Server 9.x (jira.mts.ru, 2026-09-29): the issue sent as inwardIssue is the
// one that afterwards reads with the OUTWARD wording, on both issues' pages. So
// the subject goes into inwardIssue and the object into outwardIssue.
func (c *Client) CreateIssueLink(typeName, subjectKey, objectKey string) error {
	if typeName == "" {
		return fmt.Errorf("CreateIssueLink: link type is required")
	}
	if subjectKey == "" || objectKey == "" {
		return fmt.Errorf("CreateIssueLink: both issue keys are required")
	}
	if equalFoldTrimmed(subjectKey, objectKey) {
		return fmt.Errorf("CreateIssueLink: refusing to link %s to itself", subjectKey)
	}

	req := createIssueLinkRequest{
		Type:         issueLinkTypeRef{Name: typeName},
		InwardIssue:  issueLinkRef{Key: subjectKey},
		OutwardIssue: issueLinkRef{Key: objectKey},
	}

	if _, err := c.Post(c.apiPathFor("issueLink"), &req); err != nil {
		return fmt.Errorf("CreateIssueLink %s -[%s]-> %s: %w", subjectKey, typeName, objectKey, err)
	}
	return nil
}

// equalFoldTrimmed compares two names ignoring case and surrounding whitespace.
func equalFoldTrimmed(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// IssueLink is one existing link on an issue. Exactly one of InwardIssue or
// OutwardIssue is set: the one that is set is the OTHER issue, and the side it
// sits on tells which wording this issue reads with.
type IssueLink struct {
	ID           string        `json:"id"`
	Type         IssueLinkType `json:"type"`
	InwardIssue  *LinkedIssue  `json:"inwardIssue,omitempty"`
	OutwardIssue *LinkedIssue  `json:"outwardIssue,omitempty"`
}

// LinkedIssue is the other end of a link.
type LinkedIssue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
		Status  *struct {
			Name string `json:"name"`
		} `json:"status"`
	} `json:"fields"`
}

type issueLinksResponse struct {
	Fields struct {
		IssueLinks []IssueLink `json:"issuelinks"`
	} `json:"fields"`
}

// GetIssueLinks returns the links already on an issue.
func (c *Client) GetIssueLinks(issueKey string) ([]IssueLink, error) {
	q := urlValuesWithFields("issuelinks")
	data, err := c.Get(c.apiPathFor("issue", issueKey), q)
	if err != nil {
		return nil, fmt.Errorf("GetIssueLinks %s: %w", issueKey, err)
	}

	var resp issueLinksResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("GetIssueLinks %s: failed to unmarshal: %w", issueKey, err)
	}
	return resp.Fields.IssueLinks, nil
}

// DeleteIssueLink removes a link by its id, so a wrong link is reversible.
func (c *Client) DeleteIssueLink(linkID string) error {
	if linkID == "" {
		return fmt.Errorf("DeleteIssueLink: link id is required")
	}
	if _, err := c.Delete(c.apiPathFor("issueLink", linkID)); err != nil {
		return fmt.Errorf("DeleteIssueLink %s: %w", linkID, err)
	}
	return nil
}

// urlValuesWithFields builds a query asking Jira for just the named fields.
func urlValuesWithFields(fields string) url.Values {
	q := url.Values{}
	q.Set("fields", fields)
	return q
}
