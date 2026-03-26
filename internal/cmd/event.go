package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/basecamp/hey-cli/internal/output"
)

type eventCommand struct {
	cmd *cobra.Command
}

func newEventCommand() *eventCommand {
	eventCommand := &eventCommand{}
	eventCommand.cmd = &cobra.Command{
		Use:   "event",
		Short: "Manage calendar events",
		Annotations: map[string]string{
			"agent_notes": "Subcommands: list, create, edit, delete. Use list --ids-only to pipe IDs to edit/delete.",
		},
	}

	eventCommand.cmd.AddCommand(newEventListCommand().cmd)

	return eventCommand
}

// list

type eventListCommand struct {
	cmd   *cobra.Command
	limit int
	all   bool
}

func newEventListCommand() *eventListCommand {
	c := &eventListCommand{}
	c.cmd = &cobra.Command{
		Use:   "list",
		Short: "List calendar events",
		Example: `  hey event list
  hey event list --limit 10`,
		RunE: c.run,
	}

	c.cmd.Flags().IntVar(&c.limit, "limit", 0, "Maximum number of events to show")
	c.cmd.Flags().BoolVar(&c.all, "all", false, "Fetch all results (override --limit)")

	return c
}

func (c *eventListCommand) run(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}

	ctx := cmd.Context()
	resp, err := listPersonalRecordings(ctx)
	if err != nil {
		return err
	}

	events := filterRecordingsByType(resp, "Calendar::Event")

	total := len(events)
	if c.limit > 0 && !c.all && len(events) > c.limit {
		events = events[:c.limit]
	}
	notice := output.TruncationNotice(len(events), total)

	if writer.IsStyled() {
		if len(events) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No events.")
			return nil
		}

		table := newTable(cmd.OutOrStdout())
		table.addRow([]string{"ID", "Title", "Date", "Start", "End", "All Day"})
		for _, e := range events {
			allDay := ""
			if e.AllDay {
				allDay = "yes"
			}
			table.addRow([]string{
				fmt.Sprintf("%d", e.Id),
				e.Title,
				formatDate(e.StartsAt),
				formatTimestamp(e.StartsAt),
				formatTimestamp(e.EndsAt),
				allDay,
			})
		}
		table.print()
		if notice != "" {
			fmt.Fprintln(cmd.OutOrStdout(), notice)
		}
		return nil
	}

	return writeOK(events,
		output.WithSummary(fmt.Sprintf("%d events", len(events))),
		output.WithNotice(notice),
		output.WithBreadcrumbs(
			output.Breadcrumb{
				Action:      "create",
				Command:     "hey event create '...' --date YYYY-MM-DD --start HH:MM --end HH:MM",
				Description: "Create a new event",
			},
			output.Breadcrumb{
				Action:      "delete",
				Command:     "hey event delete <id>",
				Description: "Delete an event",
			},
		),
	)
}
