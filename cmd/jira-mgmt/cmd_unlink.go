package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var unlinkCmd = &cobra.Command{
	Use:   "unlink <LINK-ID...>",
	Short: "Remove issue links by id",
	Long: `Remove issue links by their ids.

Link ids come from 'jira-mgmt link <ISSUE-KEY> --show'.

Example:
  jira-mgmt unlink 123456`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := buildJiraClientFromConfig()
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		for _, id := range args {
			if err := client.DeleteIssueLink(id); err != nil {
				return err
			}
			fmt.Fprintf(out, "link %s removed\n", id)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unlinkCmd)
}
