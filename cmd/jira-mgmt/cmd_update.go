package main

import (
	"fmt"
	"strings"

	"github.com/relux-works/skill-jira-management/internal/jira"
	"github.com/spf13/cobra"
)

var (
	updateSummary     string
	updateDescription string
)

var (
	updateFields      []string
	updateClearFields []string
)

var updateCmd = &cobra.Command{
	Use:   "update ISSUE-KEY",
	Short: "Update issue fields (summary, description)",
	Long: `Update fields on an existing issue.

Examples:
  jira-mgmt update PROJ-123 --summary "New title"
  jira-mgmt update PROJ-123 --description "New description"
  jira-mgmt update PROJ-123 --summary "New title" --description "New description"
  jira-mgmt update PROJ-123 --field "Story Points=5"
  jira-mgmt update PROJ-123 --clear-field "Smart Checklist"

--field and --clear-field address any field the instance defines, custom ones
included, by name or by id. 'jira-mgmt fields <KEY>' lists them with their ids.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		issueKey := args[0]
		out := cmd.OutOrStdout()

		if updateSummary == "" && updateDescription == "" && len(updateFields) == 0 && len(updateClearFields) == 0 {
			return fmt.Errorf("at least one of --summary, --description, --field or --clear-field is required")
		}

		client, err := buildJiraClientFromConfig()
		if err != nil {
			return err
		}

		fields := make(map[string]interface{})
		if updateSummary != "" {
			fields["summary"] = updateSummary
		}
		if updateDescription != "" {
			// Server/DC v2 accepts plain string, Cloud v3 needs ADF.
			if client.IsCloud() {
				fields["description"] = jira.NewADFText(updateDescription)
			} else {
				fields["description"] = updateDescription
			}
		}

		// Resolve every named field before writing any of them: a typo must not
		// leave the issue half-updated, and writing to the wrong custom field is
		// silent.
		for _, spec := range updateFields {
			name, value, found := strings.Cut(spec, "=")
			if !found || strings.TrimSpace(name) == "" {
				return fmt.Errorf("--field %q must be NAME=VALUE", spec)
			}
			def, err := client.ResolveFieldID(strings.TrimSpace(name))
			if err != nil {
				return err
			}
			fields[def.ID] = value
		}
		for _, name := range updateClearFields {
			def, err := client.ResolveFieldID(strings.TrimSpace(name))
			if err != nil {
				return err
			}
			fields[def.ID] = nil
		}

		req := &jira.UpdateIssueRequest{Fields: fields}
		if err := client.UpdateIssue(issueKey, req); err != nil {
			return fmt.Errorf("updating %s: %w", issueKey, err)
		}

		fmt.Fprintf(out, "Updated %s\n", issueKey)
		return nil
	},
}

func init() {
	updateCmd.Flags().StringVar(&updateSummary, "summary", "", "New issue summary/title")
	updateCmd.Flags().StringVar(&updateDescription, "description", "", "New issue description")
	updateCmd.Flags().StringArrayVar(&updateFields, "field", nil, "Set a field: NAME=VALUE (repeatable; name or id)")
	updateCmd.Flags().StringArrayVar(&updateClearFields, "clear-field", nil, "Clear a field by name or id (repeatable)")
	rootCmd.AddCommand(updateCmd)
}
