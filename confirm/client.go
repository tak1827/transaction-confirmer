package confirm

import (
	"context"
)

type Client interface {
	SendTx(ctx context.Context, tx interface{}) (string, error)
	ConfirmTx(ctx context.Context, hash string, confirmationBlocks uint64) error
	// Nonce(ctx context.Context, privKey string) (nonce uint64, err error)
	// LatestBlockNumber(ctx context.Context) (uint64, error)
}
