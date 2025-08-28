package cmd

import (
	"context"
	"fmt"

	"github.com/onkernel/cli/pkg/util"
	"github.com/onkernel/kernel-go-sdk"
	"github.com/onkernel/kernel-go-sdk/option"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// ProfilesService defines the subset of the Kernel SDK profile client that we use.
// Mirrors patterns from BrowsersService.
type ProfilesService interface {
	Get(ctx context.Context, idOrName string, opts ...option.RequestOption) (res *kernel.Profile, err error)
	List(ctx context.Context, opts ...option.RequestOption) (res *[]kernel.Profile, err error)
	Delete(ctx context.Context, idOrName string, opts ...option.RequestOption) (err error)
	New(ctx context.Context, body kernel.ProfileNewParams, opts ...option.RequestOption) (res *kernel.Profile, err error)
}

type ProfilesGetInput struct {
	Identifier string
}

type ProfilesCreateInput struct {
	Name string
}

type ProfilesDeleteInput struct {
	Identifier  string
	SkipConfirm bool
}

// ProfilesCmd handles profile operations independent of cobra.
type ProfilesCmd struct {
	profiles ProfilesService
}

func (p ProfilesCmd) List(ctx context.Context) error {
	pterm.Info.Println("Fetching profiles...")
	items, err := p.profiles.List(ctx)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if items == nil || len(*items) == 0 {
		pterm.Info.Println("No profiles found")
		return nil
	}
	rows := pterm.TableData{{"Profile ID", "Name", "Created At", "Updated At", "Last Used At"}}
	for _, prof := range *items {
		name := prof.Name
		if name == "" {
			name = "-"
		}
		rows = append(rows, []string{prof.ID, name, prof.CreatedAt.Format("2006-01-02 15:04:05"), prof.UpdatedAt.Format("2006-01-02 15:04:05"), prof.LastUsedAt.Format("2006-01-02 15:04:05")})
	}
	printTableNoPad(rows, true)
	return nil
}

func (p ProfilesCmd) Get(ctx context.Context, in ProfilesGetInput) error {
	item, err := p.profiles.Get(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if item == nil || item.ID == "" {
		pterm.Error.Printf("Profile '%s' not found\n", in.Identifier)
		return nil
	}
	name := item.Name
	if name == "" {
		name = "-"
	}
	rows := pterm.TableData{{"Property", "Value"}}
	rows = append(rows, []string{"ID", item.ID})
	rows = append(rows, []string{"Name", name})
	rows = append(rows, []string{"Created At", item.CreatedAt.Format("2006-01-02 15:04:05")})
	rows = append(rows, []string{"Updated At", item.UpdatedAt.Format("2006-01-02 15:04:05")})
	printTableNoPad(rows, true)
	return nil
}

func (p ProfilesCmd) Create(ctx context.Context, in ProfilesCreateInput) error {
	params := kernel.ProfileNewParams{}
	if in.Name != "" {
		params.Name = kernel.Opt(in.Name)
	}
	item, err := p.profiles.New(ctx, params)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	name := item.Name
	if name == "" {
		name = "-"
	}
	rows := pterm.TableData{{"Property", "Value"}}
	rows = append(rows, []string{"ID", item.ID})
	rows = append(rows, []string{"Name", name})
	rows = append(rows, []string{"Created At", item.CreatedAt.Format("2006-01-02 15:04:05")})
	rows = append(rows, []string{"Updated At", item.UpdatedAt.Format("2006-01-02 15:04:05")})
	printTableNoPad(rows, true)
	return nil
}

func (p ProfilesCmd) Delete(ctx context.Context, in ProfilesDeleteInput) error {
	if !in.SkipConfirm {
		// Try to resolve for a nicer message; avoid prompting for missing entries
		list, err := p.profiles.List(ctx)
		if err != nil {
			return util.CleanedUpSdkError{Err: err}
		}
		var found *kernel.Profile
		if list != nil {
			for _, pr := range *list {
				if pr.ID == in.Identifier || (pr.Name != "" && pr.Name == in.Identifier) {
					cp := pr
					found = &cp
					break
				}
			}
		}
		if found == nil {
			pterm.Error.Printf("Profile '%s' not found\n", in.Identifier)
			return nil
		}
		// Confirm
		msg := fmt.Sprintf("Are you sure you want to delete profile '%s'?", in.Identifier)
		pterm.DefaultInteractiveConfirm.DefaultText = msg
		ok, _ := pterm.DefaultInteractiveConfirm.Show()
		if !ok {
			pterm.Info.Println("Deletion cancelled")
			return nil
		}
	}

	if err := p.profiles.Delete(ctx, in.Identifier); err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	pterm.Success.Printf("Deleted profile: %s\n", in.Identifier)
	return nil
}

// --- Cobra wiring ---

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Manage profiles",
	Long:  "Commands for managing Kernel browser profiles",
}

var profilesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List profiles",
	RunE:  runProfilesList,
}

var profilesGetCmd = &cobra.Command{
	Use:   "get <id-or-name>",
	Short: "Get a profile by ID or name",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfilesGet,
}

var profilesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new profile",
	RunE:  runProfilesCreate,
}

var profilesDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-name>",
	Short: "Delete a profile by ID or name",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfilesDelete,
}

func init() {
	profilesCmd.AddCommand(profilesListCmd)
	profilesCmd.AddCommand(profilesGetCmd)
	profilesCmd.AddCommand(profilesCreateCmd)
	profilesCmd.AddCommand(profilesDeleteCmd)

	profilesCreateCmd.Flags().String("name", "", "Optional unique profile name")
	profilesDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}

func runProfilesList(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Profiles
	p := ProfilesCmd{profiles: &svc}
	return p.List(cmd.Context())
}

func runProfilesGet(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Profiles
	p := ProfilesCmd{profiles: &svc}
	return p.Get(cmd.Context(), ProfilesGetInput{Identifier: args[0]})
}

func runProfilesCreate(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	name, _ := cmd.Flags().GetString("name")
	svc := client.Profiles
	p := ProfilesCmd{profiles: &svc}
	return p.Create(cmd.Context(), ProfilesCreateInput{Name: name})
}

func runProfilesDelete(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	skip, _ := cmd.Flags().GetBool("yes")
	svc := client.Profiles
	p := ProfilesCmd{profiles: &svc}
	return p.Delete(cmd.Context(), ProfilesDeleteInput{Identifier: args[0], SkipConfirm: skip})
}
