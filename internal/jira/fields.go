package jira

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// FieldDefinition is one field the instance knows about, custom or built in.
type FieldDefinition struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Name   string `json:"name"`
	Custom bool   `json:"custom"`
}

// ListFieldDefinitions returns every field the instance defines. Custom fields
// are addressed by an opaque id (customfield_10042), so a caller that only knows
// the human name needs this to translate.
func (c *Client) ListFieldDefinitions() ([]FieldDefinition, error) {
	data, err := c.Get(c.apiPathFor("field"), nil)
	if err != nil {
		return nil, fmt.Errorf("ListFieldDefinitions: %w", err)
	}

	var defs []FieldDefinition
	if err := json.Unmarshal(data, &defs); err != nil {
		return nil, fmt.Errorf("ListFieldDefinitions: failed to unmarshal: %w", err)
	}
	return defs, nil
}

// ResolveFieldID turns a field name or id into the id the API expects. An exact
// id match wins; otherwise the name is matched case-insensitively. A name that
// matches several fields is refused rather than guessed, because writing to the
// wrong custom field is silent and hard to notice.
func (c *Client) ResolveFieldID(nameOrID string) (FieldDefinition, error) {
	defs, err := c.ListFieldDefinitions()
	if err != nil {
		return FieldDefinition{}, err
	}

	var matches []FieldDefinition
	for _, d := range defs {
		if d.ID == nameOrID {
			return d, nil
		}
		if equalFoldTrimmed(d.Name, nameOrID) {
			matches = append(matches, d)
		}
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return FieldDefinition{}, fmt.Errorf("no field named %q on this instance", nameOrID)
	default:
		ids := make([]string, 0, len(matches))
		for _, m := range matches {
			ids = append(ids, m.ID)
		}
		return FieldDefinition{}, fmt.Errorf("field name %q is ambiguous: %s; address it by id", nameOrID, strings.Join(ids, ", "))
	}
}

// GetIssueRawFields returns the issue's fields exactly as the API renders them,
// so a caller can see custom fields the typed model does not carry.
func (c *Client) GetIssueRawFields(issueKey string) (map[string]json.RawMessage, error) {
	q := url.Values{}
	q.Set("fields", "*all")

	data, err := c.Get(c.apiPathFor("issue", issueKey), q)
	if err != nil {
		return nil, fmt.Errorf("GetIssueRawFields %s: %w", issueKey, err)
	}

	var envelope struct {
		Fields map[string]json.RawMessage `json:"fields"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("GetIssueRawFields %s: failed to unmarshal: %w", issueKey, err)
	}
	return envelope.Fields, nil
}

// SetIssueFields writes the given fields by id. A nil value clears a field.
func (c *Client) SetIssueFields(issueKey string, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return fmt.Errorf("SetIssueFields %s: no fields given", issueKey)
	}
	for id := range fields {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("SetIssueFields %s: a field id is empty", issueKey)
		}
	}

	body := map[string]interface{}{"fields": fields}
	if _, err := c.Put(c.apiPathFor("issue", issueKey), &body); err != nil {
		return fmt.Errorf("SetIssueFields %s: %w", issueKey, err)
	}
	return nil
}

// RawRequest calls a REST path as given, for endpoints outside /rest/api that
// the typed methods do not model -- plugin APIs above all. The body is passed
// through unparsed so the caller controls the exact shape.
func (c *Client) RawRequest(method, path, body string) ([]byte, error) {
	var payload interface{}
	if body != "" {
		payload = json.RawMessage(body)
	}
	data, err := c.request(method, path, nil, payload)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	return data, nil
}
