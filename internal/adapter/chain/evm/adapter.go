package evm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/hdwallet"
	"github.com/karolchmiel94/omnicatena/internal/port"
)

var nativeETH = domain.Asset{Symbol: "ETH", Decimals: 18, Native: true}

type Config struct {
	RPCURL  string
	ChainID int64
	Chain   domain.ChainID
}

type Adapter struct {
	cfg    Config
	client *ethclient.Client
	signer types.Signer
}

func New(cfg Config) (*Adapter, error) {
	client, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("evm: dial %s: %w", cfg.RPCURL, err)
	}
	return &Adapter{
		cfg:    cfg,
		client: client,
		signer: types.NewLondonSigner(big.NewInt(cfg.ChainID)),
	}, nil
}

func (a *Adapter) Chain() domain.ChainID { return a.cfg.Chain }

func (a *Adapter) DeriveAccount(seed []byte, path domain.DerivationPath) (domain.Account, error) {
	key, err := hdwallet.DeriveKey(seed, path)
	if err != nil {
		return domain.Account{}, err
	}
	privKey, err := key.ECPrivKey()
	if err != nil {
		return domain.Account{}, fmt.Errorf("evm: ec priv key: %w", err)
	}
	addr := crypto.PubkeyToAddress(privKey.ToECDSA().PublicKey)
	return domain.Account{
		Chain:   a.cfg.Chain,
		Address: domain.Address(addr.Hex()),
		Path:    path,
	}, nil
}

func (a *Adapter) Balance(ctx context.Context, addr domain.Address, _ domain.Asset) (domain.Amount, error) {
	bal, err := a.client.BalanceAt(ctx, common.HexToAddress(string(addr)), nil)
	if err != nil {
		return domain.Amount{}, fmt.Errorf("evm: balance %s: %w", addr, err)
	}
	return domain.Amount{Asset: nativeETH, Base: bal}, nil
}

func (a *Adapter) EstimateFee(ctx context.Context, req domain.TransferRequest) (domain.FeeEstimate, error) {
	tip, err := a.client.SuggestGasTipCap(ctx)
	if err != nil {
		return domain.FeeEstimate{}, fmt.Errorf("evm: suggest tip: %w", err)
	}
	head, err := a.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return domain.FeeEstimate{}, fmt.Errorf("evm: latest header: %w", err)
	}
	baseFee := head.BaseFee
	if baseFee == nil {
		baseFee = big.NewInt(0)
	}

	adjustedTip := ScaleTip(tip, req.Speed)
	// standard EIP-1559 formula: maxFee = 2*baseFee + tip
	maxFee := new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), adjustedTip)

	const gasLimit = 21_000
	totalWei := new(big.Int).Mul(maxFee, big.NewInt(gasLimit))

	return domain.FeeEstimate{
		Speed: req.Speed,
		Total: domain.Amount{Asset: nativeETH, Base: totalWei},
		Params: map[string]string{
			"gas_limit":                fmt.Sprintf("%d", gasLimit),
			"max_fee_per_gas":          maxFee.String(),
			"max_priority_fee_per_gas": adjustedTip.String(),
		},
	}, nil
}

func (a *Adapter) BuildTransfer(ctx context.Context, req domain.TransferRequest) (domain.UnsignedTx, error) {
	from := common.HexToAddress(string(req.From))
	to := common.HexToAddress(string(req.To))

	nonce, err := a.client.PendingNonceAt(ctx, from)
	if err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("evm: nonce: %w", err)
	}
	fee, err := a.EstimateFee(ctx, req)
	if err != nil {
		return domain.UnsignedTx{}, err
	}

	maxFee := new(big.Int)
	maxFee.SetString(fee.Params["max_fee_per_gas"], 10)
	tip := new(big.Int)
	tip.SetString(fee.Params["max_priority_fee_per_gas"], 10)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(a.cfg.ChainID),
		Nonce:     nonce,
		GasTipCap: tip,
		GasFeeCap: maxFee,
		Gas:       21_000,
		To:        &to,
		Value:     req.Amount.Base,
	})
	payload, err := tx.MarshalBinary()
	if err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("evm: marshal tx: %w", err)
	}
	return domain.UnsignedTx{
		Chain:   a.cfg.Chain,
		Request: req,
		Fee:     fee,
		Payload: payload,
	}, nil
}

func (a *Adapter) Sign(_ context.Context, utx domain.UnsignedTx, signer port.Signer) (domain.SignedTx, error) {
	var tx types.Transaction
	if err := tx.UnmarshalBinary(utx.Payload); err != nil {
		return domain.SignedTx{}, fmt.Errorf("evm: unmarshal tx: %w", err)
	}
	hash := a.signer.Hash(&tx)
	sig, err := signer.Sign(utx.Account, hash[:])
	if err != nil {
		return domain.SignedTx{}, fmt.Errorf("evm: sign: %w", err)
	}
	signedTx, err := tx.WithSignature(a.signer, sig)
	if err != nil {
		return domain.SignedTx{}, fmt.Errorf("evm: attach signature: %w", err)
	}
	raw, err := signedTx.MarshalBinary()
	if err != nil {
		return domain.SignedTx{}, fmt.Errorf("evm: marshal signed tx: %w", err)
	}
	return domain.SignedTx{
		Chain: a.cfg.Chain,
		Raw:   raw,
		Hash:  signedTx.Hash().Hex(),
	}, nil
}

func (a *Adapter) Broadcast(ctx context.Context, tx domain.SignedTx) (string, error) {
	var signedTx types.Transaction
	if err := signedTx.UnmarshalBinary(tx.Raw); err != nil {
		return "", fmt.Errorf("evm: unmarshal: %w", err)
	}
	if err := a.client.SendTransaction(ctx, &signedTx); err != nil {
		return "", fmt.Errorf("evm: broadcast: %w", err)
	}
	return signedTx.Hash().Hex(), nil
}

func (a *Adapter) GetTransaction(ctx context.Context, hash string) (domain.Transaction, error) {
	h := common.HexToHash(hash)
	tx, _, err := a.client.TransactionByHash(ctx, h)
	if err != nil {
		if err == ethereum.NotFound {
			return domain.Transaction{Chain: a.cfg.Chain, Hash: hash, Status: domain.TxUnknown}, nil
		}
		return domain.Transaction{}, fmt.Errorf("evm: get tx %s: %w", hash, err)
	}

	receipt, err := a.client.TransactionReceipt(ctx, h)
	if err != nil {
		// tx is known but not yet mined
		_ = tx
		return domain.Transaction{Chain: a.cfg.Chain, Hash: hash, Status: domain.TxPending}, nil
	}

	status := domain.TxConfirmed
	if receipt.Status == types.ReceiptStatusFailed {
		status = domain.TxFailed
	}

	gasPrice := receipt.EffectiveGasPrice
	if gasPrice == nil {
		gasPrice = tx.GasPrice()
	}
	feeWei := new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), gasPrice)

	return domain.Transaction{
		Chain:       a.cfg.Chain,
		Hash:        hash,
		Status:      status,
		BlockHeight: receipt.BlockNumber.Uint64(),
		Fee:         domain.Amount{Asset: nativeETH, Base: feeWei},
	}, nil
}

// ScaleTip adjusts the suggested miner tip by speed: fast=1.5×, economy=0.8×, standard=1×.
func ScaleTip(tip *big.Int, speed domain.FeeSpeed) *big.Int {
	mul := int64(10)
	switch speed {
	case domain.SpeedFast:
		mul = 15
	case domain.SpeedEconomy:
		mul = 8
	}
	return new(big.Int).Div(new(big.Int).Mul(tip, big.NewInt(mul)), big.NewInt(10))
}
