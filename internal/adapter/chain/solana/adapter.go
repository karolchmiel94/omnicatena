package solana

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	bin "github.com/gagliardetto/binary"
	solanago "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/hdwallet"
	"github.com/karolchmiel94/omnicatena/internal/port"
	"crypto/ed25519"
)

var nativeSOL = domain.Asset{Symbol: "SOL", Decimals: 9, Native: true}

// baseFee is the fixed per-signature fee on Solana (5000 lamports).
const baseFee = uint64(5000)

type Adapter struct {
	client *rpc.Client
}

func New(rpcURL string) *Adapter {
	return &Adapter{client: rpc.New(rpcURL)}
}

func (a *Adapter) Chain() domain.ChainID { return domain.ChainSolana }

func (a *Adapter) DeriveAccount(seed []byte, path domain.DerivationPath) (domain.Account, error) {
	keySeed, err := hdwallet.DeriveKeyEd25519(seed, path)
	if err != nil {
		return domain.Account{}, err
	}
	priv := ed25519.NewKeyFromSeed(keySeed)
	pub := priv.Public().(ed25519.PublicKey)
	pk := solanago.PublicKeyFromBytes(pub)
	return domain.Account{
		Chain:   domain.ChainSolana,
		Address: domain.Address(pk.String()),
		Path:    path,
	}, nil
}

func (a *Adapter) Balance(ctx context.Context, addr domain.Address, _ domain.Asset) (domain.Amount, error) {
	pk, err := solanago.PublicKeyFromBase58(string(addr))
	if err != nil {
		return domain.Amount{}, fmt.Errorf("solana: invalid address %s: %w", addr, err)
	}
	resp, err := a.client.GetBalance(ctx, pk, rpc.CommitmentConfirmed)
	if err != nil {
		return domain.Amount{}, fmt.Errorf("solana: balance %s: %w", addr, err)
	}
	return domain.Amount{Asset: nativeSOL, Base: new(big.Int).SetUint64(resp.Value)}, nil
}

func (a *Adapter) EstimateFee(_ context.Context, req domain.TransferRequest) (domain.FeeEstimate, error) {
	return domain.FeeEstimate{
		Speed: req.Speed,
		Total: domain.Amount{Asset: nativeSOL, Base: new(big.Int).SetUint64(baseFee)},
		Params: map[string]string{
			"lamports": fmt.Sprintf("%d", baseFee),
		},
	}, nil
}

func (a *Adapter) BuildTransfer(ctx context.Context, req domain.TransferRequest) (domain.UnsignedTx, error) {
	from, err := solanago.PublicKeyFromBase58(string(req.From))
	if err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("solana: invalid from address: %w", err)
	}
	to, err := solanago.PublicKeyFromBase58(string(req.To))
	if err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("solana: invalid to address: %w", err)
	}

	blockhashResp, err := a.client.GetLatestBlockhash(ctx, rpc.CommitmentConfirmed)
	if err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("solana: get blockhash: %w", err)
	}

	instr := system.NewTransferInstruction(req.Amount.Base.Uint64(), from, to).Build()
	tx, err := solanago.NewTransaction(
		[]solanago.Instruction{instr},
		blockhashResp.Value.Blockhash,
		solanago.TransactionPayer(from),
	)
	if err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("solana: build tx: %w", err)
	}

	payload, err := tx.MarshalBinary()
	if err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("solana: serialize tx: %w", err)
	}

	fee, _ := a.EstimateFee(ctx, req)
	return domain.UnsignedTx{
		Chain:   domain.ChainSolana,
		Request: req,
		Fee:     fee,
		Payload: payload,
	}, nil
}

func (a *Adapter) Sign(_ context.Context, utx domain.UnsignedTx, signer port.Signer) (domain.SignedTx, error) {
	tx, err := solanago.TransactionFromDecoder(bin.NewBinDecoder(utx.Payload))
	if err != nil {
		return domain.SignedTx{}, fmt.Errorf("solana: decode tx: %w", err)
	}

	msgBytes, err := tx.Message.MarshalBinary()
	if err != nil {
		return domain.SignedTx{}, fmt.Errorf("solana: marshal message: %w", err)
	}

	sig, err := signer.Sign(utx.Account, msgBytes)
	if err != nil {
		return domain.SignedTx{}, fmt.Errorf("solana: sign: %w", err)
	}

	var solSig solanago.Signature
	copy(solSig[:], sig)
	tx.Signatures = []solanago.Signature{solSig}

	raw, err := tx.MarshalBinary()
	if err != nil {
		return domain.SignedTx{}, fmt.Errorf("solana: marshal signed tx: %w", err)
	}

	return domain.SignedTx{
		Chain: domain.ChainSolana,
		Raw:   raw,
		Hash:  solSig.String(),
	}, nil
}

func (a *Adapter) Broadcast(ctx context.Context, stx domain.SignedTx) (string, error) {
	tx, err := solanago.TransactionFromDecoder(bin.NewBinDecoder(stx.Raw))
	if err != nil {
		return "", fmt.Errorf("solana: decode signed tx: %w", err)
	}
	sig, err := a.client.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{
		PreflightCommitment: rpc.CommitmentConfirmed,
	})
	if err != nil {
		return "", fmt.Errorf("solana: broadcast: %w", err)
	}
	return sig.String(), nil
}

func (a *Adapter) GetTransaction(ctx context.Context, hash string) (domain.Transaction, error) {
	sig, err := solanago.SignatureFromBase58(hash)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("solana: invalid signature %s: %w", hash, err)
	}

	maxVersion := uint64(0)
	result, err := a.client.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
		Encoding:                       solanago.EncodingBase64,
		Commitment:                     rpc.CommitmentConfirmed,
		MaxSupportedTransactionVersion: &maxVersion,
	})
	if err != nil {
		if errors.Is(err, rpc.ErrNotFound) {
			return domain.Transaction{Chain: domain.ChainSolana, Hash: hash, Status: domain.TxUnknown}, nil
		}
		return domain.Transaction{}, fmt.Errorf("solana: get tx %s: %w", hash, err)
	}

	if result.Meta.Err != nil {
		return domain.Transaction{
			Chain:  domain.ChainSolana,
			Hash:   hash,
			Status: domain.TxFailed,
		}, nil
	}

	return domain.Transaction{
		Chain:       domain.ChainSolana,
		Hash:        hash,
		Status:      domain.TxConfirmed,
		BlockHeight: result.Slot,
		Fee:         domain.Amount{Asset: nativeSOL, Base: new(big.Int).SetUint64(result.Meta.Fee)},
	}, nil
}
