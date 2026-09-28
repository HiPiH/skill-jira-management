package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	fieldsShowAll bool
	fieldsOnly    string
)

var fieldsCmd = &cobra.Command{
	Use:   "fields <ISSUE-KEY>",
	Short: "Show an issue's fields, including custom ones",
	Long: `Show every field an issue carries, custom fields included.

The typed read commands cover the standard fields; this one is for the rest --
Smart Checklist, story points, and whatever else the instance defines. The id is
what 'update --field' and '--clear-field' address.

Examples:
  jira-mgmt fields PROJ-123
  jira-mgmt fields PROJ-123 --all`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := buildJiraClientFromConfig()
		if err != nil {
			return err
		}

		raw, err := client.GetIssueRawFields(args[0])
		if err != nil {
			return err
		}
		defs, err := client.ListFieldDefinitions()
		if err != nil {
			return err
		}
		names := map[string]string{}
		custom := map[string]bool{}
		for _, d := range defs {
			names[d.ID] = d.Name
			custom[d.ID] = d.Custom
		}

		if fieldsOnly != "" {
			def, err := client.ResolveFieldID(fieldsOnly)
			if err != nil {
				return err
			}
			value, ok := raw[def.ID]
			if !ok {
				return fmt.Errorf("issue %s carries no field %s (%s)", args[0], def.ID, def.Name)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s (%s):\n%s\n", def.ID, def.Name, string(value))
			return nil
		}

		ids := make([]string, 0, len(raw))
		for id := range raw {
			ids = append(ids, id)
		}
		sort.Strings(ids)

		out := cmd.OutOrStdout()
		shown := 0
		for _, id := range ids {
			value := strings.TrimSpace(string(raw[id]))
			if !fieldsShowAll && (value == "" || value == "null" || value == "[]" || value == "{}") {
				continue
			}
			kind := " "
			if custom[id] {
				kind = "*"
			}
			fmt.Fprintf(out, "%s %-22s %-34s %s\n", kind, id, names[id], preview(value))
			shown++
		}
		if shown == 0 {
			fmt.Fprintln(out, "(no fields with a value)")
		}
		fmt.Fprintln(out, "\n* = custom field. Clear one with: jira-mgmt update <KEY> --clear-field <id-or-name>")
		return nil
	},
}

// preview keeps one field to one line: the value is for recognising the field,
// not for reading its contents.
func preview(value string) string {
	var pretty interface{}
	if err := json.Unmarshal([]byte(value), &pretty); err == nil {
		if s, ok := pretty.(string); ok {
			value = s
		}
	}
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 90 {
		return value[:90] + "…"
	}
	return value
}

func init() {
	fieldsCmd.Flags().BoolVar(&fieldsShowAll, "all", false, "Include fields that are empty")
	fieldsCmd.Flags().StringVar(&fieldsOnly, "field", "", "Print one field in full, by id or name")

	rootCmd.AddCommand(fieldsCmd)
}
