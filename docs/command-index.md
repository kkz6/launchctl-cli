# Command index

Generated from the CLI command tree. Use `lctl <command> --help` for argument syntax and defaults.

All commands inherit `--json`, `--ci`, `--profile`, and `--api-url`; individual commands may remain interactive or use plain output. Flags on parent groups also apply to their subcommands. See the workflow guides for behavior and limitations.

| Command | Purpose | Local flags |
| --- | --- | --- |
| `lctl ai` | Manage the operate-launchctl AI skill | `--codex-home` |
| `lctl ai doctor` | Check the bundled AI skill installation |  |
| `lctl ai install` | Install the bundled AI skill for Codex |  |
| `lctl ai uninstall` | Remove an lctl-managed AI skill installation | `--force` |
| `lctl ai update` | Update an lctl-managed AI skill installation | `--force` |
| `lctl api` | Call an authenticated launchctl API endpoint | `--data` |
| `lctl auth` | Sign in through your browser using a one-time code | `--no-browser` |
| `lctl auth login` | Sign in through your browser using a one-time code | `--no-browser` |
| `lctl auth status` | Show authentication status |  |
| `lctl completion` | Generate the autocompletion script for the specified shell |  |
| `lctl completion bash` | Generate the autocompletion script for bash | `--no-descriptions` |
| `lctl completion fish` | Generate the autocompletion script for fish | `--no-descriptions` |
| `lctl completion powershell` | Generate the autocompletion script for powershell | `--no-descriptions` |
| `lctl completion zsh` | Generate the autocompletion script for zsh | `--no-descriptions` |
| `lctl config` | Manage CLI configuration |  |
| `lctl config profiles` | Manage configuration profiles |  |
| `lctl config profiles add` | Add a new profile |  |
| `lctl config profiles remove` | Remove a profile |  |
| `lctl config profiles use` | Switch to a profile |  |
| `lctl config set` | Set a configuration value |  |
| `lctl config show` | Show current configuration |  |
| `lctl cron` | Manage cron jobs |  |
| `lctl cron add` | Add a cron job | `--server` |
| `lctl cron delete` | Delete a cron job | `--server` |
| `lctl cron list` | List cron jobs | `--server` |
| `lctl daemons` | Manage daemons |  |
| `lctl daemons add` | Add a daemon | `--server` |
| `lctl daemons delete` | Delete a daemon | `--server` |
| `lctl daemons list` | List daemons | `--server` |
| `lctl daemons restart` | Restart a daemon | `--server` |
| `lctl databases` | Manage server databases |  |
| `lctl databases create` | Create a new database | `--server` |
| `lctl databases delete` | Delete a database | `--server` |
| `lctl databases list` | List databases on a server | `--server` |
| `lctl databases users` | List database users on a server | `--server` |
| `lctl deploy` | Deploy sites and manage deployments |  |
| `lctl deploy cancel-queued` | Cancel queued deployments for a site | `--server`, `--yes` |
| `lctl deploy list` | List deployments for a site | `--server` |
| `lctl deploy logs` | View deployment logs | `--follow`, `--server` |
| `lctl deploy rollback` | Roll back to a previous deployment | `--server`, `--site`, `--timeout`, `--wait`, `--yes` |
| `lctl deploy show` | Show deployment details | `--server`, `--site` |
| `lctl deploy trigger` | Trigger a deployment | `--server`, `--timeout`, `--wait` |
| `lctl docker` | Manage Docker projects and applications |  |
| `lctl docker applications` | Manage Docker applications | `--project`, `--server` |
| `lctl docker applications create` | Create a Docker application | `--branch`, `--build-location`, `--build-type`, `--dockerfile-path`, `--dockerfile`, `--image`, `--name`, `--port`, `--registry-credential`, `--repo`, `--source-control`, `--source` |
| `lctl docker applications delete` | Delete a Docker application | `--remove-volumes`, `--yes` |
| `lctl docker applications deploy` | Deploy or rebuild a Docker application | `--timeout`, `--wait` |
| `lctl docker applications deployments` | List Docker application deployments |  |
| `lctl docker applications list` | List applications in a Docker project |  |
| `lctl docker applications reload` | Recreate a Docker application with its saved runtime configuration |  |
| `lctl docker applications show` | Show Docker application details |  |
| `lctl docker applications start` | Start a stopped Docker application |  |
| `lctl docker applications stop` | Stop a Docker application |  |
| `lctl docker applications update` | Update a Docker application | `--build-type`, `--dockerfile-path`, `--name` |
| `lctl docker composes` | Manage Docker Compose stacks | `--project`, `--server` |
| `lctl docker composes create` | Register a stack from a local Compose YAML file | `--file`, `--name` |
| `lctl docker composes delete` | Delete a Compose stack | `--remove-volumes`, `--yes` |
| `lctl docker composes deploy` | Deploy or rebuild a Compose stack | `--timeout`, `--wait` |
| `lctl docker composes deployments` | List Compose deployment history |  |
| `lctl docker composes list` | List stacks in a Docker project |  |
| `lctl docker composes reload` | Recreate a stack with saved settings without rebuilding | `--yes` |
| `lctl docker composes show` | Show a Compose stack | `--show-config` |
| `lctl docker composes update` | Update a stack's name or saved runtime settings | `--env-file`, `--name`, `--run-command` |
| `lctl docker projects` | Manage Docker projects | `--server` |
| `lctl docker projects create` | Create a Docker project | `--description`, `--name` |
| `lctl docker projects delete` | Delete an empty Docker project | `--yes` |
| `lctl docker projects list` | List Docker projects on a server |  |
| `lctl docker projects show` | Show Docker project details |  |
| `lctl docker projects update` | Update a Docker project | `--description`, `--name` |
| `lctl env` | Manage environment files |  |
| `lctl env pull` | Pull .env file from server | `--output`, `--server`, `--site` |
| `lctl env push` | Push .env file to server | `--file`, `--server`, `--site` |
| `lctl events` | Watch live team and resource events | `--channel`, `--filter` |
| `lctl firewall` | Manage firewall rules |  |
| `lctl firewall add` | Add a firewall rule | `--server` |
| `lctl firewall delete` | Delete a firewall rule | `--server` |
| `lctl firewall list` | List firewall rules | `--server` |
| `lctl help` | Help about any command |  |
| `lctl init` | Initialize project configuration | `--application`, `--project`, `--server`, `--site` |
| `lctl login` | Sign in through your browser using a one-time code | `--no-browser` |
| `lctl logout` | Log out and clear credentials |  |
| `lctl logs` | View server and site logs | `--follow`, `--lines`, `--server`, `--site`, `--type` |
| `lctl run` | Execute a command on a remote site | `--history`, `--server`, `--site` |
| `lctl servers` | Manage servers |  |
| `lctl servers archive` | Archive the server record without stopping the server | `--yes` |
| `lctl servers create` | Register a custom server or provision a cloud server | `--credential-id`, `--database-type`, `--description`, `--install-agent`, `--ip`, `--name`, `--os`, `--php-version`, `--provider`, `--region`, `--size`, `--ssh-key`, `--ssh-port`, `--ssh-user`, `--type`, `--yes` |
| `lctl servers create-options` | Show supported server settings and provider plans as JSON |  |
| `lctl servers list` | List all servers |  |
| `lctl servers metrics` | Show latest server metrics | `--watch` |
| `lctl servers reboot` | Reboot the server, causing brief downtime | `--yes` |
| `lctl servers retry-provision` | Retry provisioning for a failed server | `--yes` |
| `lctl servers show` | Show server details |  |
| `lctl servers ssh` | Open a terminal session on a server | `--user` |
| `lctl servers try-connection` | Check a custom server's SSH connection and start provisioning | `--yes` |
| `lctl servers unarchive` | Restore an archived server record |  |
| `lctl servers update` | Update settings for a provisioned server | `--auto-update`, `--description`, `--monitoring`, `--name`, `--ssh-port` |
| `lctl servers watch` | Watch server provisioning and lifecycle events |  |
| `lctl services` | Manage server services |  |
| `lctl services list` | List installed services on a server | `--server` |
| `lctl services restart` | Restart a service | `--server` |
| `lctl services start` | Start a service | `--server` |
| `lctl services stop` | Stop a service | `--server` |
| `lctl sites` | Manage sites |  |
| `lctl sites auto-deploy` | Enable or disable automatic deployments |  |
| `lctl sites auto-deploy disable` | disable automatic deployments for a site | `--server` |
| `lctl sites auto-deploy enable` | enable automatic deployments for a site | `--server` |
| `lctl sites create` | Create a site and request its initial deployment | `--address`, `--branch`, `--php-version`, `--repository-id`, `--server`, `--source-control`, `--type`, `--web-folder`, `--zero-downtime` |
| `lctl sites delete` | Request deletion of a site | `--server`, `--yes` |
| `lctl sites list` | List sites on a server | `--server` |
| `lctl sites show` | Show site details | `--server` |
| `lctl sites update` | Update site settings | `--branch`, `--php-version`, `--queue-deployments`, `--release-retention`, `--server`, `--web-folder` |
| `lctl ssh-keys` | Manage SSH keys |  |
| `lctl ssh-keys add` | Add an SSH key | `--global`, `--key`, `--name` |
| `lctl ssh-keys attach` | Attach an SSH key to a server | `--server` |
| `lctl ssh-keys delete` | Delete an SSH key |  |
| `lctl ssh-keys detach` | Detach an SSH key from a server | `--server` |
| `lctl ssh-keys list` | List team SSH keys |  |
| `lctl ssh-keys server-list` | List SSH keys attached to a server | `--server` |
| `lctl ssl` | Manage SSL certificates |  |
| `lctl ssl list` | List SSL certificates | `--server`, `--site` |
| `lctl status` | Show dashboard overview |  |
| `lctl switch` | Switch to a different profile |  |
| `lctl tasks` | Inspect and watch server tasks | `--server` |
| `lctl tasks list` | List recent tasks for a server |  |
| `lctl tasks watch` | Watch task output and progress in real time |  |
| `lctl teams` | Manage teams |  |
| `lctl teams list` | List your teams |  |
| `lctl teams members` | List team members | `--team` |
| `lctl teams switch` | Switch active team |  |
| `lctl update` | Check for and install lctl updates | `--check`, `--force` |
| `lctl version` | Print the version of lctl |  |
| `lctl whoami` | Show current user and team info |  |
