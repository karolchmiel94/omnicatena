package cli

import (
	"fmt"
	"time"

	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/spf13/cobra"
)

func (c *CLI) walletCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "wallet", Short: "Manage wallets"}
	cmd.AddCommand(c.walletCreate(), c.walletList(), c.walletGet(), c.walletBalance())
	return cmd
}

func (c *CLI) walletCreate() *cobra.Command {
	var label string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new HD wallet",
		RunE: func(cmd *cobra.Command, args []string) error {
			pass, err := readPassphrase()
			if err != nil {
				return err
			}
			w, err := c.wallets.Create(cmd.Context(), label, pass)
			if err != nil {
				return err
			}
			if c.jsonOut {
				return c.printJSON(w)
			}
			printWallet(w)
			return nil
		},
	}
	cmd.Flags().StringVar(&label, "label", "", "wallet label")
	_ = cmd.MarkFlagRequired("label")
	return cmd
}

func (c *CLI) walletList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all wallets",
		RunE: func(cmd *cobra.Command, args []string) error {
			wallets, err := c.wallets.List(cmd.Context())
			if err != nil {
				return err
			}
			if c.jsonOut {
				return c.printJSON(wallets)
			}
			for i, w := range wallets {
				if i > 0 {
					fmt.Println()
				}
				printWallet(w)
			}
			return nil
		},
	}
}

func (c *CLI) walletGet() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a wallet by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := c.wallets.Get(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if c.jsonOut {
				return c.printJSON(w)
			}
			printWallet(w)
			return nil
		},
	}
}

func (c *CLI) walletBalance() *cobra.Command {
	var chain string
	cmd := &cobra.Command{
		Use:   "balance <id>",
		Short: "Get wallet balance on a chain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			amount, err := c.wallets.Balance(cmd.Context(), args[0], domain.ChainID(chain))
			if err != nil {
				return err
			}
			if c.jsonOut {
				return c.printJSON(map[string]string{
					"chain":  chain,
					"amount": amount.Base.String(),
					"asset":  amount.Asset.Symbol,
				})
			}
			fmt.Printf("%-10s %s %s\n", chain, amount.Base.String(), amount.Asset.Symbol)
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain ID (bitcoin, ethereum, ...)")
	_ = cmd.MarkFlagRequired("chain")
	return cmd
}

func printWallet(w domain.Wallet) {
	fmt.Printf("ID:      %s\n", w.ID)
	fmt.Printf("Label:   %s\n", w.Label)
	fmt.Printf("Created: %s\n", w.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Accounts:\n")
	for _, a := range w.Accounts {
		fmt.Printf("  %-10s %s\n", a.Chain, a.Address)
	}
}
