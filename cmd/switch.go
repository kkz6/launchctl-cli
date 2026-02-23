package cmd

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "Switch to a different profile",
	Long:  "Switch the active profile. If no name is given, shows an interactive picker.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles := cfg.ListProfiles()
		if len(profiles) == 0 {
			return fmt.Errorf("no profiles configured — run: lctl config profiles add <name>")
		}

		var name string

		if len(args) == 1 {
			name = args[0]
		} else {
			sort.Strings(profiles)
			active := cfg.ActiveProfileName()

			var options []huh.Option[string]
			for _, p := range profiles {
				label := p
				if p == active {
					label += " (active)"
				}
				if prof, ok := cfg.Profiles[p]; ok && prof.UserEmail != "" {
					label += " — " + prof.UserEmail
				}
				options = append(options, huh.NewOption(label, p))
			}

			err := huh.NewSelect[string]().
				Title("Switch profile").
				Options(options...).
				Value(&name).
				Run()
			if err != nil {
				return err
			}
		}

		if err := cfg.UseProfile(name); err != nil {
			return err
		}

		fmt.Println(tui.Success.Render(fmt.Sprintf("Switched to profile %q", name)))

		if p, ok := cfg.Profiles[name]; ok && p.UserEmail != "" {
			fmt.Println(tui.Dim.Render(fmt.Sprintf("  %s", p.UserEmail)))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)
}
