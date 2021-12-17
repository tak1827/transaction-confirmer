package confirmer

import (
	"context"
	"sync/atomic"
)

type Nonce struct {
	current uint64
}

func NewNonce(ctx context.Context, client Client, privKey string) (n Nonce, err error) {
	current, err := client.Nonce(ctx, privKey)
	if err != nil {
		return
	}

	n.current = current
	return
}

func (n *Nonce) Increment() uint64 {
	current := atomic.LoadUint64(&n.current)
	atomic.AddUint64(&n.current, 1)
	return current
}

func (n *Nonce) Reset(nonce uint64) {
	atomic.StoreUint64(&n.current, nonce)
}

func (n *Nonce) Current() uint64 {
	return atomic.LoadUint64(&n.current)
}
