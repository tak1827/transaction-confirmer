package confirmer

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/tak1827/go-queue/queue"
)

type Confirmer struct {
	client Client
	queue  *queue.Queue

	confirmationBlocks   uint64
	confirmationInterval int64 // sec
	workers              int
	timeout              int64 // sec

	afterTxSent      func(string) error
	afterTxConfirmed func(string) error
	errHandler       func(error)

	closeCounter uint32
}

type entry struct {
	hash      string
	updatedAt int64
}

func NewConfirmer(client Client, queueSize int, opts ...Opt) Confirmer {
	q := queue.NewQueue(queueSize, false)

	c := Confirmer{
		client:               client,
		queue:                &q,
		confirmationBlocks:   DEFAULT_CONFIEMATION_BLOCKS,
		confirmationInterval: DEFAULT_CONFIEMATION_INTERVAL,
		workers:              DEFAULT_WORKERS,
		timeout:              DEFAULT_TIMEOUT,
		afterTxSent:          DefaultAfterTxSent,
		afterTxConfirmed:     DefaultAfterTxConfirmed,
		errHandler:           DefaultErrHandler,
		closeCounter:         0,
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

	e := entry{
		hash:      hash,
		updatedAt: time.Now().Unix(),
	}

	if err = c.queue.Enqueue(e); err != nil {
		return errors.Wrap(err, "err Enqueue")
	}

	return nil
}

func (c *Confirmer) DequeueTx(ctx context.Context) error {
	v, isEmpty := c.queue.Dequeue()
	if isEmpty {
		return nil
	}

	e := v.(entry)

	if time.Now().Unix() < e.updatedAt+c.confirmationInterval {
		if err := c.queue.Enqueue(e); err != nil {
			return errors.Wrap(err, "err Enqueue")
		}
		return nil
	}

	if err := c.client.ConfirmTx(ctx, e.hash, c.confirmationBlocks); err != nil {

		if !errors.Is(err, ErrTxNotFound) && !errors.Is(err, ErrTxConfirmPending) {
			return errors.Wrap(err, "err ConfirmTx")
		}

		if err = c.queue.Enqueue(e); err != nil {
			return errors.Wrap(err, "err Enqueue")
		}
		return nil
	}

	if err := c.afterTxConfirmed(e.hash); err != nil {
		return errors.Wrap(err, "err afterTxSent")
	}

	return nil
}

func (c *Confirmer) QueueLen() int {
	return c.queue.Len()
}

func (c *Confirmer) Start() error {
	worker := func(c *Confirmer, id int) {
		for !c.closing() {
			ctx, cancel := c.withTimeout()
			defer cancel()

			if err := c.DequeueTx(ctx); err != nil {
				c.errHandler(err)
			}
		}

		fmt.Printf("worker(%d) is closing\n", id)
		atomic.AddUint32(&c.closeCounter, 1)
	}

	for i := 0; i < c.workers; i++ {
		id := i + 1
		go worker(c, id)
	}

	fmt.Print("confirmer is ready\n")
	return nil
}

func (c *Confirmer) Close() {
	atomic.AddUint32(&c.closeCounter, 1)
	for !c.closed() {
	}
	fmt.Print("confirmer is closed\n")
}

func (c *Confirmer) withTimeout() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	return context.WithTimeout(ctx, time.Duration(time.Duration(c.timeout)*time.Second))
}

func (c *Confirmer) closing() bool {
	return atomic.LoadUint32(&c.closeCounter) > 0
}

func (c *Confirmer) closed() bool {
	return atomic.LoadUint32(&c.closeCounter) >= 1+uint32(c.workers)
}
