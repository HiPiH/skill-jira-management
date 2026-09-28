package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var apiBody string

var apiCmd = &cobra.Command{
	Use:   "api <METHOD> <PATH>",
	Short: "Call a Jira REST path directly",
	Long: `Call a REST path on the configured instance with the stored credentials.

For endpoints the typed commands do not cover -- plugin APIs in particular,
which live outside /rest/api and have their own paths. The path is taken as
given, so it must start with /rest.

Examples:
  jira-mgmt api GET /rest/api/2/myself
  jira-mgmt api GET /rest/com.example.plugin/1.0/thing/PROJ-1
  jira-mgmt api PUT /rest/api/2/issue/PROJ-1 --body '{"fields":{"summary":"x"}}'`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		method := strings.ToUpper(args[0])
		path := args[1]

		if !strings.HasPrefix(path, "/rest") {
			return fmt.Errorf("path must start with /rest, got %q", path)
		}
		switch method {
		case "GET", "POST", "PUT", "DELETE":
		default:
			return fmt.Errorf("unsupported method %q", method)
		}
		if apiBody != "" && method == "GET" {
			return fmt.Errorf("GET takes no --body")
		}

		client, err := buildJiraClientFromConfig()
		if err != nil {
			return err
		}

		data, err := client.RawRequest(method, path, apiBody)
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		if len(data) == 0 {
			fmt.Fprintf(out, "%s %s: no content\n", method, path)
			return nil
		}
		fmt.Fprintln(out, string(data))
		return nil
	},
}

func init() {
	apiCmd.Flags().StringVar(&apiBody, "body", "", "Raw JSON request body")

	rootCmd.AddCommand(apiCmd)
}
