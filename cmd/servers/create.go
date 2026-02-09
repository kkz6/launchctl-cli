package servers

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		opts, err := client.GetCreateServerOptions()
		if err != nil {
			return fmt.Errorf("failed to fetch options: %w", err)
		}

		if len(opts.Providers) == 0 {
			return fmt.Errorf("no server providers configured, add one in the web dashboard first")
		}

		var name, providerIdx, serverType, os, region, size string

		providerOptions := make([]huh.Option[string], len(opts.Providers))
		for i, p := range opts.Providers {
			providerOptions[i] = huh.NewOption(p.Name, fmt.Sprintf("%d", i))
		}

		typeOptions := make([]huh.Option[string], len(opts.Types))
		for i, t := range opts.Types {
			typeOptions[i] = huh.NewOption(t.Label, t.Value)
		}

		osOptions := make([]huh.Option[string], len(opts.OperatingSystems))
		for i, o := range opts.OperatingSystems {
			osOptions[i] = huh.NewOption(o.Label, o.Value)
		}

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Server Name").
					Placeholder("my-server").
					Value(&name),
				huh.NewSelect[string]().
					Title("Provider").
					Options(providerOptions...).
					Value(&providerIdx),
				huh.NewSelect[string]().
					Title("Type").
					Options(typeOptions...).
					Value(&serverType),
				huh.NewSelect[string]().
					Title("Operating System").
					Options(osOptions...).
					Value(&os),
			),
		)

		if err := form.Run(); err != nil {
			return err
		}

		var idx int
		fmt.Sscanf(providerIdx, "%d", &idx)
		provider := opts.Providers[idx]

		regionOptions := make([]huh.Option[string], len(provider.Regions))
		for i, r := range provider.Regions {
			regionOptions[i] = huh.NewOption(r.Label, r.Value)
		}

		sizeOptions := make([]huh.Option[string], len(provider.Sizes))
		for i, s := range provider.Sizes {
			sizeOptions[i] = huh.NewOption(s.Label, s.Value)
		}

		form2 := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Region").
					Options(regionOptions...).
					Value(&region),
				huh.NewSelect[string]().
					Title("Size").
					Options(sizeOptions...).
					Value(&size),
			),
		)

		if err := form2.Run(); err != nil {
			return err
		}

		server, err := client.CreateServer(api.CreateServerRequest{
			Name:            name,
			Provider:        provider.Provider,
			CredentialID:    provider.ID,
			Type:            serverType,
			Region:          region,
			Size:            size,
			OperatingSystem: os,
		})
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		fmt.Println()
		fmt.Println(tui.Success.Render("Server created successfully"))
		fmt.Println(tui.Label.Render("ID:") + tui.Value.Render(server.ID))
		fmt.Println(tui.Label.Render("Name:") + tui.Value.Render(server.Name))
		fmt.Println(tui.Label.Render("Status:") + tui.Value.Render(server.StatusLabel))
		fmt.Println()

		return nil
	},
}
