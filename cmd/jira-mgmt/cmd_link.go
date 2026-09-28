package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	linkType      string
	linkTo        string
	linkListTypes bool
	linkReverse   bool
	linkShow      bool
)

var linkCmd = &cobra.Command{
	Use:   "link [ISSUE-KEY...]",
	Short: "Link Jira issues to each other",
	Long: `Create issue links between issues.

The named issues read with the link type's outward wording, the --to issue with
its inward wording: with type "Blocks" (outward "blocks"),

  jira-mgmt link PROJ-1 --type Blocks --to PROJ-2

reads "PROJ-1 blocks PROJ-2". --reverse swaps the two sides, which is what a
batch under one parent needs:

  jira-mgmt link PROJ-2 PROJ-3 --type Parent --to PROJ-1 --reverse

reads "PROJ-1 parent of PROJ-2" and "PROJ-1 parent of PROJ-3".

Examples:
  jira-mgmt link --list-types
  jira-mgmt link PROJ-1 --show
  jira-mgmt link PROJ-1 --type "Relates" --to PROJ-2
  jira-mgmt link PROJ-1 PROJ-2 PROJ-3 --type "Relates" --to PROJ-100`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := buildJiraClientFromConfig()
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		if linkListTypes {
			types, err := client.ListIssueLinkTypes()
			if err != nil {
				return err
			}
			fmt.Fprintln(out, "available link types:")
			for _, t := range types {
				fmt.Fprintf(out, "  %-24s outward: %-24s inward: %s\n", t.Name, t.Outward, t.Inward)
			}
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("at least one issue key is required (or use --list-types)")
		}

		if linkShow {
			for _, key := range args {
				links, err := client.GetIssueLinks(key)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "%s:\n", key)
				if len(links) == 0 {
					fmt.Fprintln(out, "  (no links)")
				}
				for _, l := range links {
					if l.OutwardIssue != nil {
						fmt.Fprintf(out, "  [%s] %s %s  %s\n", l.ID, l.Type.Outward, l.OutwardIssue.Key, l.OutwardIssue.Fields.Summary)
					}
					if l.InwardIssue != nil {
						fmt.Fprintf(out, "  [%s] %s %s  %s\n", l.ID, l.Type.Inward, l.InwardIssue.Key, l.InwardIssue.Fields.Summary)
					}
				}
			}
			return nil
		}
		if linkType == "" {
			return fmt.Errorf("--type is required (see --list-types)")
		}
		if linkTo == "" {
			return fmt.Errorf("--to is required")
		}

		// Refuse an unknown type once, before touching any issue, so a batch
		// cannot half-apply on a typo.
		resolved, err := client.ResolveIssueLinkType(linkType)
		if err != nil {
			return err
		}

		for _, key := range args {
			subjectKey, objectKey := key, linkTo
			if linkReverse {
				subjectKey, objectKey = linkTo, key
			}

			if err := client.CreateIssueLink(resolved.Name, subjectKey, objectKey); err != nil {
				return err
			}
			fmt.Fprintf(out, "%s %s %s\n", subjectKey, resolved.Outward, objectKey)
		}
		return nil
	},
}

func init() {
	linkCmd.Flags().StringVar(&linkType, "type", "", "Link type name (see --list-types)")
	linkCmd.Flags().StringVar(&linkTo, "to", "", "The issue the named issues point at (reads with the inward wording)")
	linkCmd.Flags().BoolVar(&linkListTypes, "list-types", false, "List the link types configured on the instance")
	linkCmd.Flags().BoolVar(&linkReverse, "reverse", false, "Swap the sides: --to becomes the outward issue")
	linkCmd.Flags().BoolVar(&linkShow, "show", false, "Show the links already on the named issues")

	rootCmd.AddCommand(linkCmd)
}
