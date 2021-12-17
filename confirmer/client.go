package confirmer

import (
	"context"
)

type Client interface {
	Nonce(ctx context.Context, privKey string) (nonce uint64, err error)
	SendTx(ctx context.Context, tx interface{}) (string, error)
	ConfirmTx(ctx context.Context, hash string, confirmBlock uint64) error
	LatestBlockNumber(ctx context.Context) (uint64, error)
}
