package jira

import (
	"encoding/json"
	"time"
)

// --- Client Config ---

// InstanceType represents the Jira deployment type.
type InstanceType string

const (
	InstanceCloud  InstanceType = "cloud"  // Jira Cloud (*.atlassian.net)
	InstanceServer InstanceType = "server" // Jira Server / Data Center
)

// AuthType represents the authentication method.
type AuthType string

const (
	AuthBasic  AuthType = "basic"  // Cloud: email + API token → Basic base64(email:token)
	AuthBearer AuthType = "bearer" // Server/DC: Personal Access Token → Bearer <token>
)

// Config holds the configuration needed to connect to a Jira instance.
type Config struct {
	BaseURL            string       // e.g. "https://mycompany.atlassian.net"
	Email              string       // Required for Basic auth (Cloud)
	Token              string       // API token (Basic) or PAT (Bearer)
	InstanceType       InstanceType // "cloud" or "server" — auto-detected if empty
	AuthType           AuthType     // "basic" or "bearer" — inferred from Email presence if empty
	InsecureSkipVerify bool         // Skip TLS certificate verification (corporate CAs)
}

// --- Error Types ---

// APIError represents an error response from the Jira REST API.
type APIError struct {
	StatusCode    int               `json:"-"`
	ErrorMessages []string          `json:"errorMessages,omitempty"`
	Errors        map[string]string `json:"errors,omitempty"`
}

func (e *APIError) Error() string {
	if len(e.ErrorMessages) > 0 {
		return e.ErrorMessages[0]
	}
	for field, msg := range e.Errors {
		return field + ": " + msg
	}
	return "jira: unknown API error"
}

// --- ADF (Atlassian Document Format) ---

// ADFDoc is the top-level Atlassian Document Format document.
type ADFDoc struct {
	Type    string    `json:"type"`    // always "doc"
	Version int       `json:"version"` // always 1
	Content []ADFNode `json:"content"`
}

// ADFNode represents a node in an ADF document tree.
type ADFNode struct {
	Type    string          `json:"type"`
	Content []ADFNode       `json:"content,omitempty"`
	Text    string          `json:"text,omitempty"`
	Marks   []ADFMark       `json:"marks,omitempty"`
	Attrs   json.RawMessage `json:"attrs,omitempty"`
}

// ADFMark represents a formatting mark on an ADF text node.
type ADFMark struct {
	Type  string          `json:"type"`
	Attrs json.RawMessage `json:"attrs,omitempty"`
}

// --- User ---

// User represents a Jira user.
//
// Cloud addresses a user by AccountID, Server/DC by Name; the two are not
// interchangeable and each instance rejects the other's field, so both are
// carried here and the caller picks by instance type.
type User struct {
	AccountID    string `json:"accountId,omitempty"`
	Name         string `json:"name,omitempty"`
	Key          string `json:"key,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
	Active       bool   `json:"active,omitempty"`
}

// --- Issue ---

// Issue represents a Jira issue.
type Issue struct {
	ID     string      `json:"id,omitempty"`
	Key    string      `json:"key,omitempty"`
	Self   string      `json:"self,omitempty"`
	Fields IssueFields `json:"fields"`
}

// IssueFields represents the standard fields of a Jira issue.
type IssueFields struct {
	Summary        string          `json:"summary,omitempty"`
	Description    *ADFDoc         `json:"-"`                     // custom unmarshal: ADF (Cloud v3) or string (Server v2)
	DescriptionRaw json.RawMessage `json:"description,omitempty"` // raw for flexible deserialization
	IssueType      IssueType       `json:"issuetype,omitempty"`
	Project        Project         `json:"project,omitempty"`
	Status         *Status         `json:"status,omitempty"`
	Priority       *Priority       `json:"priority,omitempty"`
	Assignee       *User           `json:"assignee,omitempty"`
	Reporter       *User           `json:"reporter,omitempty"`
	Labels         []string        `json:"labels,omitempty"`
	Parent         *Issue          `json:"parent,omitempty"`
	Subtasks       []Issue         `json:"subtasks,omitempty"`
	Created        string          `json:"created,omitempty"`
	Updated        string          `json:"updated,omitempty"`
	Attachments    []Attachment    `json:"attachment,omitempty"`
	Comments       []Comment       `json:"-"` // loaded explicitly for targeted issue reads

	// Custom fields are stored here for flexible access.
	CustomFields map[string]json.RawMessage `json:"-"`
}

// Attachment is the safe metadata Jira exposes for an issue attachment.
// Content is retained internally for authenticated downloads but is not
// projected by the agent-facing query layer.
type Attachment struct {
	ID       string `json:"id,omitempty"`
	Filename string `json:"filename,omitempty"`
	Author   *User  `json:"author,omitempty"`
	Created  string `json:"created,omitempty"`
	Size     int64  `json:"size,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Content  string `json:"content,omitempty"`
}

// DescriptionText returns the description as plain text.
// Handles both ADF (Cloud) and plain string (Server/DC).
func (f *IssueFields) DescriptionText() string {
	// If Description was set directly (not from JSON), use it.
	if f.Description != nil && f.DescriptionRaw == nil {
		return extractADFText(f.Description)
	}
	if f.DescriptionRaw == nil {
		return ""
	}
	// Try as plain string first (Server/DC v2).
	var s string
	if err := json.Unmarshal(f.DescriptionRaw, &s); err == nil {
		return s
	}
	// Try as ADF document (Cloud v3).
	var doc ADFDoc
	if err := json.Unmarshal(f.DescriptionRaw, &doc); err == nil {
		f.Description = &doc
		return extractADFText(&doc)
	}
	return string(f.DescriptionRaw)
}

// extractADFText walks an ADF document and extracts plain text.
func extractADFText(doc *ADFDoc) string {
	if doc == nil {
		return ""
	}
	var buf []byte
	for _, node := range doc.Content {
		buf = extractNodeText(node, buf)
	}
	return string(buf)
}

func extractNodeText(node ADFNode, buf []byte) []byte {
	if node.Text != "" {
		buf = append(buf, node.Text...)
	}
	for _, child := range node.Content {
		buf = extractNodeText(child, buf)
	}
	if node.Type == "paragraph" || node.Type == "heading" || node.Type == "listItem" {
		buf = append(buf, '\n')
	}
	return buf
}

// IssueType represents a Jira issue type (Epic, Story, Task, Subtask, Bug).
type IssueType struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Subtask bool   `json:"subtask,omitempty"`
}

// Priority represents an issue priority.
type Priority struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// --- Issue Create/Update ---

// CreateIssueRequest is the request body for creating an issue.
type CreateIssueRequest struct {
	Fields CreateIssueFields `json:"fields"`
}

// CreateIssueFields holds the fields for issue creation.
type CreateIssueFields struct {
	Project   ProjectRef   `json:"project"`
	IssueType IssueTypeRef `json:"issuetype"`
	Summary   string       `json:"summary"`
	// Description is ADF on Cloud v3 and a plain string on Server/DC v2, so it
	// is typed loosely and the caller supplies the right shape.
	Description interface{}  `json:"description,omitempty"`
	Assignee    *UserRef     `json:"assignee,omitempty"`
	Priority    *PriorityRef `json:"priority,omitempty"`
	Labels      []string     `json:"labels,omitempty"`
	Parent      *IssueRef    `json:"parent,omitempty"`

	// Extra allows setting custom fields.
	Extra map[string]interface{} `json:"-"`
}

// ProjectRef identifies a project by key.
type ProjectRef struct {
	Key string `json:"key,omitempty"`
	ID  string `json:"id,omitempty"`
}

// IssueTypeRef identifies an issue type by id or by name. The id is preferred:
// a localized instance may have several types sharing a name.
type IssueTypeRef struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// UserRef identifies a user by account ID.
type UserRef struct {
	AccountID string `json:"accountId"`
}

// PriorityRef identifies a priority by name or ID.
type PriorityRef struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// IssueRef identifies an issue by key.
type IssueRef struct {
	Key string `json:"key,omitempty"`
	ID  string `json:"id,omitempty"`
}

// CreateIssueResponse is the response from creating an issue.
type CreateIssueResponse struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

// UpdateIssueRequest is the request body for updating an issue.
type UpdateIssueRequest struct {
	Fields map[string]interface{} `json:"fields,omitempty"`
}

// --- Project ---

// Project represents a Jira project.
type Project struct {
	ID             string `json:"id,omitempty"`
	Key            string `json:"key,omitempty"`
	Name           string `json:"name,omitempty"`
	ProjectTypeKey string `json:"projectTypeKey,omitempty"`
	Self           string `json:"self,omitempty"`
}

// ProjectSearchResult is the paginated response from project search.
type ProjectSearchResult struct {
	MaxResults int       `json:"maxResults"`
	StartAt    int       `json:"startAt"`
	Total      int       `json:"total"`
	IsLast     bool      `json:"isLast"`
	Values     []Project `json:"values"`
}

// --- Board ---

// Board represents a Jira Agile board.
type Board struct {
	ID   int    `json:"id"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"` // "scrum" or "kanban"
	Self string `json:"self,omitempty"`
}

// BoardSearchResult is the paginated response from board listing.
type BoardSearchResult struct {
	MaxResults int     `json:"maxResults"`
	StartAt    int     `json:"startAt"`
	Total      int     `json:"total"`
	IsLast     bool    `json:"isLast"`
	Values     []Board `json:"values"`
}

// --- Sprint ---

// Sprint represents a Jira sprint.
type Sprint struct {
	ID            int        `json:"id"`
	Name          string     `json:"name,omitempty"`
	State         string     `json:"state,omitempty"` // "future", "active", "closed"
	StartDate     *time.Time `json:"startDate,omitempty"`
	EndDate       *time.Time `json:"endDate,omitempty"`
	OriginBoardID int        `json:"originBoardId,omitempty"`
	Goal          string     `json:"goal,omitempty"`
	Self          string     `json:"self,omitempty"`
}

// SprintSearchResult is the paginated response from sprint listing.
type SprintSearchResult struct {
	MaxResults int      `json:"maxResults"`
	StartAt    int      `json:"startAt"`
	IsLast     bool     `json:"isLast"`
	Values     []Sprint `json:"values"`
}

// --- Transition ---

// Transition represents a workflow transition.
type Transition struct {
	ID     string                     `json:"id"`
	Name   string                     `json:"name"`
	To     Status                     `json:"to"`
	Fields map[string]TransitionField `json:"fields,omitempty"`
}

// TransitionsResponse wraps the transitions list.
type TransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

// TransitionField describes an issue field exposed on a workflow transition screen.
type TransitionField struct {
	Required      bool                   `json:"required,omitempty"`
	Schema        *TransitionFieldSchema `json:"schema,omitempty"`
	Name          string                 `json:"name,omitempty"`
	FieldID       string                 `json:"fieldId,omitempty"`
	Operations    []string               `json:"operations,omitempty"`
	AllowedValues []TransitionOption     `json:"allowedValues,omitempty"`
}

// TransitionFieldSchema describes a transition field schema.
type TransitionFieldSchema struct {
	Type     string `json:"type,omitempty"`
	System   string `json:"system,omitempty"`
	Custom   string `json:"custom,omitempty"`
	CustomID int    `json:"customId,omitempty"`
}

// TransitionOption is an allowed option value for a transition field.
type TransitionOption struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Value    string `json:"value,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

// DoTransitionRequest is the request body for executing a transition.
type DoTransitionRequest struct {
	Transition TransitionRef          `json:"transition"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
}

// TransitionRef identifies a transition by ID.
type TransitionRef struct {
	ID string `json:"id"`
}

// --- Status ---

// Status represents a Jira workflow status.
type Status struct {
	ID             string          `json:"id,omitempty"`
	Name           string          `json:"name,omitempty"`
	StatusCategory *StatusCategory `json:"statusCategory,omitempty"`
	Self           string          `json:"self,omitempty"`
}

// StatusCategory is the high-level category (To Do, In Progress, Done).
type StatusCategory struct {
	ID        int    `json:"id,omitempty"`
	Key       string `json:"key,omitempty"`
	ColorName string `json:"colorName,omitempty"`
	Name      string `json:"name,omitempty"`
}

// --- Search (JQL) ---

// SearchRequest is the POST body for JQL search.
type SearchRequest struct {
	JQL           string   `json:"jql"`
	MaxResults    int      `json:"maxResults,omitempty"`
	Fields        []string `json:"fields,omitempty"`
	NextPageToken string   `json:"nextPageToken,omitempty"` // Cloud only (cursor pagination)
	StartAt       int      `json:"startAt,omitempty"`       // Server/DC (offset pagination)
}

// SearchResponse is the response from the JQL search endpoint.
type SearchResponse struct {
	MaxResults    int     `json:"maxResults"`
	Total         int     `json:"total,omitempty"` // Server/DC returns total count
	Issues        []Issue `json:"issues"`
	NextPageToken string  `json:"nextPageToken,omitempty"` // Cloud only
	IsLast        bool    `json:"isLast"`
}

// --- Comment ---

// Comment represents a Jira issue comment.
type Comment struct {
	ID      string          `json:"id,omitempty"`
	Self    string          `json:"self,omitempty"`
	Author  *User           `json:"author,omitempty"`
	Body    *ADFDoc         `json:"-"`
	BodyRaw json.RawMessage `json:"body,omitempty"`
	Created string          `json:"created,omitempty"`
	Updated string          `json:"updated,omitempty"`
}

// BodyText returns a comment body as plain text for both Jira Cloud ADF and
// Jira Server/Data Center string responses.
func (c *Comment) BodyText() string {
	if c.Body != nil && c.BodyRaw == nil {
		return extractADFText(c.Body)
	}
	if c.BodyRaw == nil {
		return ""
	}

	var text string
	if err := json.Unmarshal(c.BodyRaw, &text); err == nil {
		return text
	}

	var doc ADFDoc
	if err := json.Unmarshal(c.BodyRaw, &doc); err == nil {
		c.Body = &doc
		return extractADFText(&doc)
	}

	return string(c.BodyRaw)
}

// CommentsResponse is the paginated response for issue comments.
type CommentsResponse struct {
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	Comments   []Comment `json:"comments"`
}

// AddCommentRequest is the request body for adding a comment.
type AddCommentRequest struct {
	Body interface{} `json:"body"`
}

// --- Pagination helper ---

// PaginationParams holds common offset-based pagination parameters.
type PaginationParams struct {
	StartAt    int
	MaxResults int
}

// --- Issue list (by project, offset-based via search/jql) ---

// ListIssuesOptions configures issue listing.
type ListIssuesOptions struct {
	ProjectKey string
	IssueType  string
	Status     string
	Fields     []string
	MaxResults int
}
