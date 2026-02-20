package nav

import (
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/notify"
	"github.com/kkz6/launchctl/internal/tui"
)

func favoriteActions(client *api.Client, cfg *config.Config, fav config.Favorite) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Favorites", fav.SiteAddress)

		choice, err := tui.SelectFromList(
			fmt.Sprintf("★ %s", fav.SiteAddress),
			[]string{"Deploy", "SSH", "View Logs", "Remove from Favorites"},
			"Back",
		)
		if err != nil || choice == 4 {
			return
		}

		server, err := client.GetServer(fav.ServerID)
		if err != nil && choice != 3 {
			tui.ShowError(fmt.Sprintf("Failed to fetch server: %s", err))
			tui.WaitForEnter()
			continue
		}

		switch choice {
		case 0:
			site, err := client.GetSite(fav.ServerID, fav.SiteID)
			if err != nil {
				tui.ShowError(fmt.Sprintf("Failed to fetch site: %s", err))
				tui.WaitForEnter()
				continue
			}
			deploySite(client, cfg, fav.ServerID, fav.ServerName, *site)
		case 1:
			if server != nil {
				sshIntoServer(cfg, *server)
			}
		case 2:
			site, err := client.GetSite(fav.ServerID, fav.SiteID)
			if err != nil {
				tui.ShowError(fmt.Sprintf("Failed to fetch site: %s", err))
				tui.WaitForEnter()
				continue
			}
			viewDeployments(client, cfg, fav.ServerID, fav.ServerName, *site)
		case 3:
			cfg.RemoveFavorite(fav.SiteID)
			notify.Success(fmt.Sprintf("Removed %s from favorites", fav.SiteAddress))
			return
		}
	}
}
