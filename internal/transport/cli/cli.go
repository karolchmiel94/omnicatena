package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/karolchmiel94/omnicatena/internal/app"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type CLI struct {
	wallets *app.WalletService
	txs     *app.TransactionService
	jsonOut bool
}

func New(w *app.WalletService, t *app.TransactionService) *CLI {
	return &CLI{wallets: w, txs: t}
}

func (c *CLI) Run(args []string) error {
	root := &cobra.Command{
		Use:           "omni",
		Short:         "Omnicatena wallet CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().BoolVar(&c.jsonOut, "json", false, "output as JSON")
	root.AddCommand(c.walletCmd(), c.txCmd())
	root.SetArgs(args)
	return root.Execute()
}

func (c *CLI) printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func readPassphrase() ([]byte, error) {
	fmt.Fprint(os.Stderr, "Passphrase: ")
	pass, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return pass, err
}
