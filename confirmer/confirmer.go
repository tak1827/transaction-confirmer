package confirmer

import (
	"context"

	"github.com/pkg/errors"
	"github.com/tak1827/go-queue/queue"
)

type Confirmer struct {
	client Client
	q      *queue.Queue

	confirmationBlocks   uint64
	confirmationInterval uint64 // milisec
	workers int

	afterTxSent func(string) error
}

type entry struct {
	hash string
	updatedAt int64
}

func NewConfirmer(client Client, queueSize int, opts ...Opt) Confirmer {
	q := queue.NewQueue(queueSize, false)

	c := Confirmer{
		client:               client,
		q:                    &q,
		confirmationBlocks:   DEFAULT_CONFIEMATION_BLOCKS,
		confirmationInterval: DEFAULT_CONFIEMATION_INTERVAL,
		workers:              DEFAULT_WORKERS,
	}

	for i := range opts {
		opts[i].Apply(&c)
	}

	return c
}

func (c *Confirmer) EnqueueTx(ctx context.Context, tx interface{}) error {
	hash, err := c.client.SendTx(ctx, tx)
	if err != nil {
		return errors.Wrap(err, "err SendTx")
	}

	if err = c.afterTxSent(hash); err != nil {
		return errors.Wrap(err, "err afterTxSent")
	}

	if err = c.q.Enqueue(hash); err != nil {
		return errors.Wrap(err, "err Enqueue")
	}

	return nil
}

func (c *Confirmer) DequeueTx(ctx context.Context) error {
	v, isEmpty := c.q.Dequeue()
	if isEmpty {
		return nil
	}

	hash := v.(string)
	c.client.ConfirmTx(ctx, hash, c.confirmationBlocks)
}

