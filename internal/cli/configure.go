package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/coder11125/patchwork/internal/keyring"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Store API keys and tokens in OS keychain",
	Long: `Interactively store API keys in your OS native keychain
(macOS Keychain, Linux Secret Service, Windows Credential Manager).

Keys stored this way are never written to disk or exposed via environment variables.`,
	RunE: runConfigure,
}

func init() {
	rootCmd.AddCommand(configureCmd)
}

func runConfigure(cmd *cobra.Command, args []string) error {
	if !keyring.IsAvailable() {
		return keyring.ErrNotAvailable
	}

	fmt.Println("Patchwork Configuration")
	fmt.Println("=======================")
	fmt.Println("Enter values or press enter to skip.")
	fmt.Println()

	llmKey := promptSecret("LLM API key (e.g. Anthropic, Mistral, Groq)")
	if llmKey != "" {
		provider := prompt("LLM provider for this key", "anthropic")
		if err := keyring.SetLLMAPIKey(strings.TrimSpace(provider), strings.TrimSpace(llmKey)); err != nil {
			return fmt.Errorf("save LLM API key: %w", err)
		}
		fmt.Println("LLM API key stored in OS keychain.")
	}

	gitToken := promptSecret("Git personal access token (GitHub/GitLab)")
	if gitToken != "" {
		platform := prompt("Git platform", "github")
		if err := keyring.SetGitToken(strings.TrimSpace(platform), strings.TrimSpace(gitToken)); err != nil {
			return fmt.Errorf("save Git token: %w", err)
		}
		fmt.Println("Git token stored in OS keychain.")
	}

	if llmKey == "" && gitToken == "" {
		fmt.Println("No values provided. Nothing stored.")
		return nil
	}

	fmt.Println()
	fmt.Println("Done. Patchwork will now use OS keychain for credentials.")
	fmt.Println("You can still override with PATCHWORK_LLM_API_KEY and PATCHWORK_GIT_TOKEN env vars.")
	return nil
}

func prompt(label, def string) string {
	fmt.Printf("%s [%s]: ", label, def)
	var v string
	fmt.Scanln(&v)
	if v == "" {
		return def
	}
	return v
}

func promptSecret(label string) string {
	fmt.Fprint(os.Stderr, label+": ")
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
	var v string
	fmt.Scanln(&v)
	return strings.TrimSpace(v)
}
