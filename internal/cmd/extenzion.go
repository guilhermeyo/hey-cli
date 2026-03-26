package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/basecamp/hey-cli/internal/output"
)

type extenzionCommand struct {
	cmd *cobra.Command
}

func newExtenzionCommand() *extenzionCommand {
	c := &extenzionCommand{}
	c.cmd = &cobra.Command{
		Use:     "extenzion",
		Aliases: []string{"ext"},
		Short:   "Manage email extensions",
		Annotations: map[string]string{
			"agent_notes": "Subcommands: list, create, edit, delete. Extensions are email aliases for HEY for Work domains.",
		},
	}

	c.cmd.AddCommand(newExtenzionListCommand().cmd)
	c.cmd.AddCommand(newExtenzionCreateCommand().cmd)

	return c
}

// resolveAccountID fetches the account ID from the identity endpoint.
func resolveAccountID(cmd *cobra.Command) (int64, string, error) {
	ctx := cmd.Context()
	identity, err := sdk.Identity().GetIdentity(ctx)
	if err != nil {
		return 0, "", convertSDKError(err)
	}
	return identity.PrimaryContact.AccountId, identity.PrimaryContact.EmailAddress, nil
}

// list

type extenzionListCommand struct {
	cmd *cobra.Command
}

func newExtenzionListCommand() *extenzionListCommand {
	c := &extenzionListCommand{}
	c.cmd = &cobra.Command{
		Use:   "list",
		Short: "List email extensions",
		Example: `  hey extenzion list
  hey ext list`,
		RunE: c.run,
	}

	return c
}

func (c *extenzionListCommand) run(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}

	accountID, _, err := resolveAccountID(cmd)
	if err != nil {
		return err
	}

	exts, err := apiClient.ListExtenzions(accountID)
	if err != nil {
		return err
	}

	if writer.IsStyled() {
		if len(exts) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No extensions.")
			return nil
		}

		table := newTable(cmd.OutOrStdout())
		table.addRow([]string{"ID", "Email", "Members"})
		for _, e := range exts {
			members := ""
			if len(e.Members) > 0 {
				members = fmt.Sprintf("%d member(s)", len(e.Members))
			}
			table.addRow([]string{
				fmt.Sprintf("%d", e.ID),
				e.Email,
				members,
			})
		}
		table.print()
		return nil
	}

	return writeOK(exts,
		output.WithSummary(fmt.Sprintf("%d extensions", len(exts))),
		output.WithBreadcrumbs(
			output.Breadcrumb{
				Action:      "create",
				Command:     "hey extenzion create <name> --member <email>",
				Description: "Create a new extension",
			},
			output.Breadcrumb{
				Action:      "delete",
				Command:     "hey extenzion delete <id>",
				Description: "Delete an extension",
			},
		),
	)
}

// create

type extenzionCreateCommand struct {
	cmd     *cobra.Command
	members []string
}

func newExtenzionCreateCommand() *extenzionCreateCommand {
	c := &extenzionCreateCommand{}
	c.cmd = &cobra.Command{
		Use:   "create <name>",
		Short: "Create an email extension",
		Example: `  hey extenzion create sales --member alice@example.com
  hey ext create support --member alice@example.com --member bob@example.com`,
		RunE: c.run,
		Args: usageExactOneArg(),
	}

	c.cmd.Flags().StringSliceVar(&c.members, "member", nil, "Member email address (repeatable, at least one required)")

	return c
}

func (c *extenzionCreateCommand) run(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}

	name := args[0]

	if len(c.members) == 0 {
		return output.ErrUsageHint("at least one --member is required",
			`hey extenzion create sales --member alice@example.com`)
	}

	accountID, email, err := resolveAccountID(cmd)
	if err != nil {
		return err
	}

	domain := splitEmail(email)
	fullEmail := name + "@" + domain

	_, err = apiClient.CreateExtenzion(accountID, name, c.members)
	if err != nil {
		return err
	}

	if writer.IsStyled() {
		fmt.Fprintf(cmd.OutOrStdout(), "Extension %s created.\n", fullEmail)
		return nil
	}

	return writeOK(map[string]any{"name": name, "email": fullEmail},
		output.WithSummary(fmt.Sprintf("Extension %s created", fullEmail)),
	)
}

// splitEmail returns the domain part of an email address.
func splitEmail(email string) string {
	if i := strings.Index(email, "@"); i >= 0 {
		return email[i+1:]
	}
	return email
}
