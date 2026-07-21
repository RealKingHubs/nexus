package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell autocompletion scripts for your terminal",
	Long: `⚡ NEXUS SHELL AUTOCOMPLETION
========================================================================
Generate autocompletion scripts for Nexus to enable live <TAB> suggestions
for subcommands, flags, and options in your terminal.

💡 QUICK SETUP GUIDES:

1. BASH (Linux / WSL):
   # Test in current session:
   $ source <(nexus completion bash)

   # Enable permanently across terminal restarts:
   $ echo 'source <(nexus completion bash)' >> ~/.bashrc

2. ZSH (macOS / WSL):
   # Enable permanently across terminal restarts:
   $ source <(nexus completion zsh)
   $ echo 'source <(nexus completion zsh)' >> ~/.zshrc

3. FISH:
   # Enable permanently:
   $ nexus completion fish > ~/.config/fish/completions/nexus.fish

4. POWERSHELL (Windows):
   # Test in current session:
   PS> nexus completion powershell | Out-String | Invoke-Expression
`,
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
