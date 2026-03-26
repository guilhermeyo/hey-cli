package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	generated "github.com/basecamp/hey-sdk/go/pkg/generated"

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
	eventCommand.cmd.AddCommand(newEventCreateCommand().cmd)
	eventCommand.cmd.AddCommand(newEventDeleteCommand().cmd)
	eventCommand.cmd.AddCommand(newEventEditCommand().cmd)

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

// create

type eventCreateCommand struct {
	cmd       *cobra.Command
	title     string
	date      string
	start     string
	end       string
	calendar  int64
	timezone  string
	allDay    bool
	reminders []string
}

func newEventCreateCommand() *eventCreateCommand {
	c := &eventCreateCommand{}
	c.cmd = &cobra.Command{
		Use:   "create [title]",
		Short: "Create a calendar event",
		Example: `  hey event create "Meeting" --date 2026-04-06 --start 10:00 --end 11:00
  hey event create "Holiday" --date 2026-04-06 --all-day
  hey event create "Standup" --date 2026-04-06 --start 09:00 --end 09:30 --reminder 30m --reminder 1d`,
		RunE: c.run,
		Args: cobra.MaximumNArgs(1),
	}

	c.cmd.Flags().StringVarP(&c.title, "title", "t", "", "Event title")
	c.cmd.Flags().StringVar(&c.date, "date", "", "Event date (YYYY-MM-DD)")
	c.cmd.Flags().StringVar(&c.start, "start", "", "Start time (HH:MM)")
	c.cmd.Flags().StringVar(&c.end, "end", "", "End time (HH:MM)")
	c.cmd.Flags().Int64Var(&c.calendar, "calendar", 0, "Calendar ID (default: personal)")
	c.cmd.Flags().StringVar(&c.timezone, "timezone", "", "Timezone name (default: system local)")
	c.cmd.Flags().BoolVar(&c.allDay, "all-day", false, "Create an all-day event")
	c.cmd.Flags().StringSliceVar(&c.reminders, "reminder", nil, "Reminder duration (e.g. 30m, 1h, 1d). Repeatable.")

	return c
}

func (c *eventCreateCommand) run(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}

	title := c.title
	if title != "" && len(args) > 0 {
		return output.ErrUsage("--title and positional argument are mutually exclusive")
	}
	if title == "" && len(args) > 0 {
		title = args[0]
	}
	if title == "" && !stdinIsTerminal() {
		var err error
		title, err = readStdin()
		if err != nil {
			return err
		}
	}
	if title == "" {
		return output.ErrUsageHint("title is required",
			`hey event create "Meeting" --date 2026-04-06 --start 10:00 --end 11:00`)
	}

	if c.date == "" {
		return output.ErrUsageHint("--date is required", "hey event create \"Meeting\" --date 2026-04-06 --start 10:00 --end 11:00")
	}

	if !c.allDay && (c.start == "" || c.end == "") {
		return output.ErrUsageHint("--start and --end are required (or use --all-day)",
			`hey event create "Meeting" --date 2026-04-06 --start 10:00 --end 11:00`)
	}

	// Resolve calendar ID
	calendarID := c.calendar
	if calendarID == 0 {
		ctx := cmd.Context()
		payload, err := sdk.Calendars().List(ctx)
		if err != nil {
			return convertSDKError(err)
		}
		calendars := unwrapCalendars(payload)
		calendarID, err = findPersonalCalendarID(calendars)
		if err != nil {
			return output.ErrNotFound("calendar", "personal")
		}
	}

	// Resolve timezone
	tz := c.timezone
	if tz == "" {
		tz = localTimezoneName()
	}

	// Build form values
	values, err := buildEventFormValues(title, c.date, c.start, c.end, calendarID, tz, c.allDay, c.reminders)
	if err != nil {
		return output.ErrUsage(err.Error())
	}

	resp, err := apiClient.CreateEvent(values)
	if err != nil {
		return err
	}

	id, _ := resp.ExtractID()
	data := map[string]any{"id": id, "location": resp.Location}

	if writer.IsStyled() {
		fmt.Fprintf(cmd.OutOrStdout(), "Event created. (id: %d)\n", id)
		return nil
	}

	return writeOK(data, output.WithSummary("Event created"))
}

// buildEventFormValues builds url.Values for the calendar event form.
// Returns error if any reminder duration is invalid.
func buildEventFormValues(title, date, start, end string, calendarID int64, tz string, allDay bool, reminders []string) (url.Values, error) {
	values := url.Values{}
	values.Set("calendar_event[calendar_id]", strconv.FormatInt(calendarID, 10))
	values.Set("calendar_event[summary]", title)
	values.Set("calendar_event[starts_at]", date)
	values.Set("calendar_event[ends_at]", date)

	if allDay {
		values.Set("calendar_event[all_day]", "1")
	} else {
		values.Set("calendar_event[all_day]", "0")
		values.Set("calendar_event[starts_at_time]", start+":00")
		values.Set("calendar_event[ends_at_time]", end+":00")
		values.Set("calendar_event[starts_at_time_zone_name]", tz)
		values.Set("calendar_event[ends_at_time_zone_name]", tz)
	}

	for _, r := range reminders {
		secs, err := parseReminderDuration(r)
		if err != nil {
			return nil, err
		}
		if allDay {
			values.Add("all_day_reminder_durations[]", strconv.Itoa(secs))
		} else {
			values.Add("timed_reminder_durations[]", strconv.Itoa(secs))
		}
	}

	return values, nil
}

// parseReminderDuration parses a human duration string (30m, 1h, 1d) to seconds.
func parseReminderDuration(s string) (int, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}
	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}
	switch unit {
	case 'm':
		return num * 60, nil
	case 'h':
		return num * 3600, nil
	case 'd':
		return num * 86400, nil
	default:
		return 0, fmt.Errorf("invalid duration unit %q in %s (use m, h, or d)", unit, s)
	}
}

// delete

type eventDeleteCommand struct {
	cmd *cobra.Command
}

func newEventDeleteCommand() *eventDeleteCommand {
	c := &eventDeleteCommand{}
	c.cmd = &cobra.Command{
		Use:     "delete <id>",
		Short:   "Delete a calendar event",
		Example: `  hey event delete 12345`,
		RunE:    c.run,
		Args:    usageExactOneArg(),
	}

	return c
}

func (c *eventDeleteCommand) run(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return output.ErrUsage(fmt.Sprintf("invalid event ID: %s", args[0]))
	}

	if err := apiClient.DeleteEvent(id); err != nil {
		return err
	}

	if writer.IsStyled() {
		fmt.Fprintln(cmd.OutOrStdout(), "Event deleted.")
		return nil
	}

	return writeOK(nil, output.WithSummary("Event deleted"))
}

// edit

type eventEditCommand struct {
	cmd       *cobra.Command
	title     string
	date      string
	start     string
	end       string
	timezone  string
	reminders []string
}

func newEventEditCommand() *eventEditCommand {
	c := &eventEditCommand{}
	c.cmd = &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit a calendar event",
		Example: `  hey event edit 12345 --start 14:00 --end 15:00
  hey event edit 12345 --title "New title" --date 2026-04-07
  hey event edit 12345 --reminder 1d --reminder 30m`,
		RunE: c.run,
		Args: usageExactOneArg(),
	}

	c.cmd.Flags().StringVarP(&c.title, "title", "t", "", "Event title")
	c.cmd.Flags().StringVar(&c.date, "date", "", "Event date (YYYY-MM-DD)")
	c.cmd.Flags().StringVar(&c.start, "start", "", "Start time (HH:MM)")
	c.cmd.Flags().StringVar(&c.end, "end", "", "End time (HH:MM)")
	c.cmd.Flags().StringVar(&c.timezone, "timezone", "", "Timezone name")
	c.cmd.Flags().StringSliceVar(&c.reminders, "reminder", nil, "Reminder duration (e.g. 30m, 1h, 1d). Repeatable.")

	return c
}

func (c *eventEditCommand) run(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return output.ErrUsage(fmt.Sprintf("invalid event ID: %s", args[0]))
	}

	// Fetch current event to merge with provided flags
	ctx := cmd.Context()
	resp, err := listPersonalRecordings(ctx)
	if err != nil {
		return err
	}

	events := filterRecordingsByType(resp, "Calendar::Event")
	var current *eventData
	for _, e := range events {
		if e.Id == id {
			current = extractEventData(e)
			break
		}
	}
	if current == nil {
		return output.ErrNotFound("event", args[0])
	}

	// Merge flags with current values
	if c.title != "" {
		current.title = c.title
	}
	if c.date != "" {
		current.date = c.date
	}
	if c.start != "" {
		current.start = c.start
	}
	if c.end != "" {
		current.end = c.end
	}
	if c.timezone != "" {
		current.timezone = c.timezone
	}

	values, err := buildEventFormValues(current.title, current.date, current.start, current.end, current.calendarID, current.timezone, current.allDay, c.reminders)
	if err != nil {
		return output.ErrUsage(err.Error())
	}

	if err := apiClient.UpdateEvent(id, values); err != nil {
		return err
	}

	if writer.IsStyled() {
		fmt.Fprintln(cmd.OutOrStdout(), "Event updated.")
		return nil
	}

	return writeOK(nil, output.WithSummary("Event updated"))
}

// eventData holds extracted event fields for merging in edit.
type eventData struct {
	title      string
	date       string
	start      string
	end        string
	timezone   string
	calendarID int64
	allDay     bool
}

// extractEventData extracts editable fields from a generated.Recording.
// Recording.StartsAt and EndsAt are time.Time, Calendar is a value type (not pointer).
func extractEventData(e generated.Recording) *eventData {
	d := &eventData{
		title:  e.Title,
		allDay: e.AllDay,
	}

	if !e.StartsAt.IsZero() {
		d.date = e.StartsAt.Format("2006-01-02")
		d.start = e.StartsAt.Format("15:04")
	}

	if !e.EndsAt.IsZero() {
		d.end = e.EndsAt.Format("15:04")
	}

	d.timezone = e.StartsAtTimeZone
	if d.timezone == "" {
		d.timezone = localTimezoneName()
	}

	if e.Calendar.Id != 0 {
		d.calendarID = e.Calendar.Id
	}

	return d
}

// localTimezoneName returns the IANA timezone name of the system (e.g. "America/Sao_Paulo").
// Falls back to timezone abbreviation if IANA name cannot be determined.
func localTimezoneName() string {
	// Check TZ environment variable first
	if tz := os.Getenv("TZ"); tz != "" {
		return tz
	}

	// Try /etc/timezone (Debian/Ubuntu)
	if data, err := os.ReadFile("/etc/timezone"); err == nil {
		if tz := strings.TrimSpace(string(data)); tz != "" {
			return tz
		}
	}

	// Try Go's Location name (works if TZ was set or system is configured)
	zone := time.Now().Location().String()
	if zone != "" && zone != "Local" {
		return zone
	}

	// Fallback to abbreviation (e.g. "BRT")
	name, _ := time.Now().Zone()
	return name
}
