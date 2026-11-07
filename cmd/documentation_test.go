package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// This is a documentation contract: the published index must match commands
// and flags registered by the executable, rather than a separate hand-kept list.
func TestCommandReference(t *testing.T) {
	rootCmd.InitDefaultCompletionCmd()
	rootCmd.InitDefaultHelpCmd()
	var rows []string
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		if cmd.Hidden {
			return
		}
		if cmd != rootCmd {
			var flags []string
			cmd.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
				if !f.Hidden && f.Deprecated == "" && f.Name != "help" {
					flags = append(flags, "`--"+f.Name+"`")
				}
			})
			cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
				if !f.Hidden && f.Deprecated == "" {
					flags = append(flags, "`--"+f.Name+"`")
				}
			})
			sort.Strings(flags)
			rows = append(rows, fmt.Sprintf("| `%s` | %s | %s |", cmd.CommandPath(), strings.ReplaceAll(cmd.Short, "|", "\\|"), strings.Join(flags, ", ")))
		}
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(rootCmd)
	body := "# Command index\n\nGenerated from the CLI command tree. Use `lctl <command> --help` for argument syntax and defaults.\n\nAll commands inherit `--json`, `--ci`, `--profile`, and `--api-url`; individual commands may remain interactive or use plain output. Flags on parent groups also apply to their subcommands. See the workflow guides for behavior and limitations.\n\n| Command | Purpose | Local flags |\n| --- | --- | --- |\n" + strings.Join(rows, "\n") + "\n"
	paths := map[string]string{filepath.Join("..", "docs", "command-index.md"): body}
	if docsDir := os.Getenv("LCTL_DOCS_DIR"); docsDir != "" {
		paths[filepath.Join(docsDir, "content", "docs", "cli", "command-index.mdx")] = "---\ntitle: Command index\ndescription: Commands and flags validated against the lctl source\n---\n\n" + body
	}
	for path, want := range paths {
		if os.Getenv("UPDATE_CLI_DOCS") == "1" {
			if err := os.WriteFile(path, []byte(want), 0644); err != nil {
				t.Fatal(err)
			}
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Errorf("%s does not match the command tree; run UPDATE_CLI_DOCS=1 LCTL_DOCS_DIR=<docs-repo> go test ./cmd -run TestCommandReference", path)
		}
	}
}

func TestDocumentedCommandExamples(t *testing.T) {
	rootCmd.InitDefaultCompletionCmd()
	rootCmd.InitDefaultHelpCmd()
	paths := []string{filepath.Join("..", "README.md"), filepath.Join("..", "docs", "cli-reference.md")}
	roots := []string{filepath.Join("..", "plugins", "launchctl", "skills", "operate-launchctl")}
	if docsDir := os.Getenv("LCTL_DOCS_DIR"); docsDir != "" {
		roots = append(roots, filepath.Join(docsDir, "content", "docs", "cli"))
	}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && (strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".mdx")) {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	inline := regexp.MustCompile("`(lctl [^`\n]+)`")
	longFlag := regexp.MustCompile(`(?:^|\s|\[)(--[a-z][a-z0-9-]*)`)
	quotedValue := regexp.MustCompile(`"[^"\n]*"|'[^'\n]*'`)
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ReplaceAll(string(data), "\\\n", " ")
		var examples []string
		for _, match := range inline.FindAllStringSubmatch(text, -1) {
			examples = append(examples, match[1])
		}
		inCode := false
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "```") {
				inCode = !inCode
				continue
			}
			if inCode && strings.HasPrefix(line, "lctl ") {
				examples = append(examples, line)
			}
		}
		for _, example := range examples {
			// Quoted command arguments can contain another program's flags.
			// Validate lctl's syntax without treating those as lctl options.
			syntax := quotedValue.ReplaceAllString(example, "ARG")
			words := strings.Fields(syntax)[1:]
			if len(words) == 0 || strings.HasPrefix(words[0], "<") {
				continue
			}
			command := rootCmd
			for index := 0; index < len(words); index++ {
				word := words[index]
				if strings.HasPrefix(word, "--") {
					name := strings.TrimPrefix(strings.SplitN(word, "=", 2)[0], "--")
					flag := command.Flags().Lookup(name)
					if flag == nil {
						flag = command.InheritedFlags().Lookup(name)
					}
					if flag == nil {
						flag = command.PersistentFlags().Lookup(name)
					}
					if flag != nil && flag.NoOptDefVal == "" && !strings.Contains(word, "=") {
						index++
					}
					continue
				}
				var next *cobra.Command
				for _, child := range command.Commands() {
					if child.Name() == word || child.HasAlias(word) {
						next = child
						break
					}
				}
				if next != nil {
					command = next
					continue
				}
				if (command == rootCmd || !command.Runnable()) && !strings.HasPrefix(word, "<") && !strings.ContainsAny(word, "/[.…") {
					t.Errorf("%s documents unknown command in %q", path, example)
				}
				break
			}
			for _, match := range longFlag.FindAllStringSubmatch(syntax, -1) {
				name := strings.TrimPrefix(match[1], "--")
				if name == "help" {
					continue
				}
				if command.Flags().Lookup(name) == nil && command.InheritedFlags().Lookup(name) == nil && command.PersistentFlags().Lookup(name) == nil {
					t.Errorf("%s documents unknown %s flag in %q", path, match[1], example)
				}
			}
		}
	}
}
