package main

import (
	"fmt"

	"github.com/relux-works/skill-jira-management/internal/jira"
	"github.com/spf13/cobra"
)

var (
	createType        string
	createSummary     string
	createDescription string
	createProjectFlag string
	createParent      string
	createLabels      []string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Jira issue",
	Long: `Create a new issue in Jira.

Examples:
  jira-mgmt create --type epic --summary "Auth system" --project PROJ
  jira-mgmt create --type story --summary "Login flow" --parent PROJ-1
  jira-mgmt create --type task --summary "Write tests" --description "Unit tests for auth"
  jira-mgmt create --type subtask --summary "Fix login" --parent PROJ-42`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := buildJiraClientFromConfig()
		if err != nil {
			return err
		}

		project := createProjectFlag
		if project == "" {
			project = flagProject
		}
		if project == "" {
			return fmt.Errorf("project required: use --project flag or configure via 'jira-mgmt config set project KEY'")
		}

		if createSummary == "" {
			return fmt.Errorf("--summary is required")
		}
		if createType == "" {
			return fmt.Errorf("--type is required (epic, story, task, subtask, bug)")
		}

		// Resolve the type against the instance: a localized Jira names its types
		// in its own language, so the English shorthand is an alias to try, not
		// the answer. The raw value wins when the instance knows it, which lets
		// a caller name an exact type ("Подзадача на разработку") or its id.
		issueType, err := client.ResolveIssueType(createType)
		if err != nil {
			alias := normalizeIssueType(createType)
			resolved, aliasErr := client.ResolveIssueType(alias)
			if aliasErr != nil {
				return fmt.Errorf("%w (also tried %q)", err, alias)
			}
			issueType = resolved
		}

		req := &jira.CreateIssueRequest{
			Fields: jira.CreateIssueFields{
				Project:   jira.ProjectRef{Key: project},
				IssueType: jira.IssueTypeRef{ID: issueType.ID},
				Summary:   createSummary,
				Labels:    createLabels,
			},
		}

		if createDescription != "" {
			// Server/DC v2 takes a plain string and refuses ADF outright
			// ("Значение операции должно быть строкой"); Cloud v3 needs ADF.
			if client.IsCloud() {
				req.Fields.Description = jira.NewADFText(createDescription)
			} else {
				req.Fields.Description = createDescription
			}
		}
		if createParent != "" {
			req.Fields.Parent = &jira.IssueRef{Key: createParent}
		}

		resp, err := client.CreateIssue(req)
		if err != nil {
			return fmt.Errorf("creating issue: %w", err)
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Created %s: %s/browse/%s\n", resp.Key, client.BaseURL(), resp.Key)
		return nil
	},
}

// normalizeIssueType converts common short names to Jira issue type names.
func normalizeIssueType(t string) string {
	switch t {
	case "epic":
		return "Epic"
	case "story":
		return "Story"
	case "task":
		return "Task"
	case "subtask", "sub-task":
		return "Sub-task"
	case "bug":
		return "Bug"
	default:
		return t
	}
}

func init() {
	createCmd.Flags().StringVar(&createType, "type", "", "Issue type: epic, story, task, subtask, bug")
	createCmd.Flags().StringVar(&createSummary, "summary", "", "Issue summary (required)")
	createCmd.Flags().StringVar(&createDescription, "description", "", "Issue description")
	createCmd.Flags().StringVar(&createProjectFlag, "project", "", "Project key (overrides global --project)")
	createCmd.Flags().StringVar(&createParent, "parent", "", "Parent issue key (for stories/subtasks)")
	createCmd.Flags().StringSliceVar(&createLabels, "label", nil, "Issue labels (repeatable)")

	rootCmd.AddCommand(createCmd)
}
