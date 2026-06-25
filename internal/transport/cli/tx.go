package cli

import (
	"fmt"
	"math/big"

	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/spf13/cobra"
)

func (c *CLI) txCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "tx", Short: "Manage transactions"}
	cmd.AddCommand(c.txTransfer(), c.txStatus())
	return cmd
}

func (c *CLI) txTransfer() *cobra.Command {
	var (
		walletID string
		chain    string
		from     string
		to       string
		amount   string
		speed    string
		env      string
	)
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Send a native asset transfer",
		RunE: func(cmd *cobra.Command, args []string) error {
			pass, err := readPassphrase()
			if err != nil {
				return err
			}
			amt := new(big.Int)
			if _, ok := amt.SetString(amount, 10); !ok {
				return fmt.Errorf("invalid amount: %q", amount)
			}
			networkEnv := domain.EnvLocal
			if env != "" {
				networkEnv = domain.NetworkEnv(env)
			}
			feeSpeed := domain.SpeedStandard
			if speed != "" {
				feeSpeed = domain.FeeSpeed(speed)
			}
			hash, err := c.txs.Transfer(cmd.Context(), walletID, pass, domain.TransferRequest{
				Network: domain.Network{Chain: domain.ChainID(chain), Env: networkEnv},
				From:    domain.Address(from),
				To:      domain.Address(to),
				// Asset.Native=true is the V1 default; add --token/--decimals flags here for V2.
				Amount: domain.Amount{Base: amt, Asset: domain.Asset{Native: true}},
				Speed:  feeSpeed,
			})
			if err != nil {
				return err
			}
			if c.jsonOut {
				return c.printJSON(map[string]string{"hash": hash})
			}
			fmt.Printf("hash  %s\n", hash)
			return nil
		},
	}
	cmd.Flags().StringVar(&walletID, "wallet", "", "wallet ID")
	cmd.Flags().StringVar(&chain, "chain", "", "chain ID")
	cmd.Flags().StringVar(&from, "from", "", "sender address")
	cmd.Flags().StringVar(&to, "to", "", "recipient address")
	cmd.Flags().StringVar(&amount, "amount", "", "amount in smallest unit (satoshi, wei, ...)")
	cmd.Flags().StringVar(&speed, "speed", "", "fee speed: economy|standard|fast (default: standard)")
	cmd.Flags().StringVar(&env, "env", "", "network env: local|testnet|mainnet (default: local)")
	for _, f := range []string{"wallet", "chain", "from", "to", "amount"} {
		_ = cmd.MarkFlagRequired(f)
	}
	return cmd
}

func (c *CLI) txStatus() *cobra.Command {
	var chain string
	cmd := &cobra.Command{
		Use:   "status <hash>",
		Short: "Get transaction status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tx, err := c.txs.Status(cmd.Context(), domain.ChainID(chain), args[0])
			if err != nil {
				return err
			}
			if c.jsonOut {
				return c.printJSON(tx)
			}
			printTx(tx)
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain ID")
	_ = cmd.MarkFlagRequired("chain")
	return cmd
}

func printTx(tx domain.Transaction) {
	fmt.Printf("chain          %s\n", tx.Chain)
	fmt.Printf("hash           %s\n", tx.Hash)
	fmt.Printf("status         %s\n", tx.Status)
	fmt.Printf("confirmations  %d\n", tx.Confirmations)
	if tx.BlockHeight > 0 {
		fmt.Printf("block          %d\n", tx.BlockHeight)
	}
}
