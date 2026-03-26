package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/basecamp/hey-cli/internal/output"
)

var validBoxKinds = []string{"imbox", "feedbox", "asidebox", "laterbox", "trailbox", "bubblebox"}

type moveCommand struct {
	cmd       *cobra.Command
	contactID int64
	box       string
	yes       bool
}

func newMoveCommand() *moveCommand {
	c := &moveCommand{}
	c.cmd = &cobra.Command{
		Use:   "move [topic-id] --box <box-kind>",
		Short: "Move a contact to a different box",
		Long:  "Designates a contact to a box. All emails from that contact will appear in the specified box.",
		Example: `  hey move 1912351860 --box feedbox
  hey move --contact 166563294 --box trailbox
  hey move 1912351860 --box imbox --yes`,
		Annotations: map[string]string{
			"agent_notes": "Moves a contact to a different box (imbox, feedbox, asidebox, laterbox, trailbox, bubblebox). Accepts topic ID or --contact. Use --yes to skip confirmation.",
		},
		RunE: c.run,
		Args: cobra.MaximumNArgs(1),
	}

	c.cmd.Flags().Int64Var(&c.contactID, "contact", 0, "Contact ID (alternative to topic ID)")
	c.cmd.Flags().StringVar(&c.box, "box", "", "Target box kind (imbox, feedbox, asidebox, laterbox, trailbox, bubblebox)")
	c.cmd.Flags().BoolVar(&c.yes, "yes", false, "Skip confirmation prompt")
	_ = c.cmd.MarkFlagRequired("box")

	return c
}

func (c *moveCommand) run(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}

	// Validate mutually exclusive args
	if c.contactID != 0 && len(args) > 0 {
		return output.ErrUsage("topic ID and --contact are mutually exclusive")
	}
	if c.contactID == 0 && len(args) == 0 {
		return output.ErrUsageHint("topic ID or --contact is required",
			"hey move 1912351860 --box feedbox  or  hey move --contact 166563294 --box feedbox")
	}

	// Validate box kind
	if !isValidBoxKind(c.box) {
		return output.ErrUsage(fmt.Sprintf("invalid box kind %q (valid: %v)", c.box, validBoxKinds))
	}

	ctx := cmd.Context()

	// Resolve contact from topic if needed
	contactID := c.contactID
	contactName := ""
	contactEmail := ""

	if contactID == 0 {
		topicID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return output.ErrUsage(fmt.Sprintf("invalid topic ID: %s", args[0]))
		}

		topic, err := sdk.Topics().Get(ctx, topicID)
		if err != nil {
			return convertSDKError(err)
		}

		contactID = topic.Creator.Id
		contactName = topic.Creator.Name
		contactEmail = topic.Creator.EmailAddress
	}

	// Resolve box ID
	boxesPtr, err := sdk.Boxes().List(ctx)
	if err != nil {
		return convertSDKError(err)
	}

	var boxID int64
	var boxName string
	if boxesPtr != nil {
		for _, b := range *boxesPtr {
			if b.Kind == c.box {
				boxID = b.Id
				boxName = b.Name
				break
			}
		}
	}
	if boxID == 0 {
		return output.ErrNotFound("box", c.box)
	}

	// Confirmation
	if !c.yes && !writer.IsStyled() {
		return output.ErrUsageHint("--yes is required in JSON mode",
			"hey move 1912351860 --box feedbox --yes --json")
	}
	if !c.yes && writer.IsStyled() {
		prompt := fmt.Sprintf("Move contact")
		if contactName != "" {
			prompt = fmt.Sprintf("Move contact %q (%s)", contactName, contactEmail)
		} else {
			prompt = fmt.Sprintf("Move contact %d", contactID)
		}
		prompt += fmt.Sprintf(" to %s? All emails from this contact will go to this box.", boxName)

		if !confirmAction(prompt) {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}
	}

	// Execute
	if err := apiClient.DesignateContact(boxID, contactID); err != nil {
		return err
	}

	if writer.IsStyled() {
		if contactName != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Contact %q (%s) moved to %s.\n", contactName, contactEmail, boxName)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Contact %d moved to %s.\n", contactID, boxName)
		}
		return nil
	}

	return writeOK(
		map[string]any{"contact_id": contactID, "box": c.box},
		output.WithSummary(fmt.Sprintf("Contact moved to %s", boxName)),
	)
}

func isValidBoxKind(kind string) bool {
	for _, k := range validBoxKinds {
		if k == kind {
			return true
		}
	}
	return false
}
