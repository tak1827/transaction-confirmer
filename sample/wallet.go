package main

import (
	"context"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
)

type Wallet struct {
	nonce   *Nonce
	priv    *ecdsa.PrivateKey
	privStr string
}

func NewWallet(ctx context.Context, client *Client, privKey string) (w Wallet, err error) {
	w.nonce, err = NewNonce(ctx, client, privKey)
	if err != nil {
		return
	}

	w.privStr = privKey

	w.priv, err = crypto.HexToECDSA(privKey)
	return
}
