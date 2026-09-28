package jira

import (
	"encoding/json"
	"fmt"
	"strings"
)

// IssueTypeDefinition is one issue type the instance offers.
type IssueTypeDefinition struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Subtask bool   `json:"subtask"`
}

// ListIssueTypes returns every issue type defined on the instance.
func (c *Client) ListIssueTypes() ([]IssueTypeDefinition, error) {
	data, err := c.Get(c.apiPathFor("issuetype"), nil)
	if err != nil {
		return nil, fmt.Errorf("ListIssueTypes: %w", err)
	}

	var types []IssueTypeDefinition
	if err := json.Unmarshal(data, &types); err != nil {
		return nil, fmt.Errorf("ListIssueTypes: failed to unmarshal: %w", err)
	}
	return types, nil
}

// ResolveIssueType turns an id or a type name into the instance's own type. A
// localized instance names its types in its own language, so the English
// shorthand the CLI accepts is not necessarily present: the caller needs to be
// able to say what the instance actually calls it.
//
// A name shared by several types is refused with their ids rather than guessed.
func (c *Client) ResolveIssueType(nameOrID string) (IssueTypeDefinition, error) {
	types, err := c.ListIssueTypes()
	if err != nil {
		return IssueTypeDefinition{}, err
	}

	var matches []IssueTypeDefinition
	for _, t := range types {
		if t.ID == nameOrID {
			return t, nil
		}
		if equalFoldTrimmed(t.Name, nameOrID) {
			matches = append(matches, t)
		}
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return IssueTypeDefinition{}, fmt.Errorf("no issue type %q on this instance", nameOrID)
	default:
		ids := make([]string, 0, len(matches))
		for _, m := range matches {
			ids = append(ids, m.ID)
		}
		return IssueTypeDefinition{}, fmt.Errorf("issue type name %q is ambiguous: %s; address it by id", nameOrID, strings.Join(ids, ", "))
	}
}
