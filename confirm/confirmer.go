package confirm

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/tak1827/go-queue/queue"
)

type (
	HashHandler func(string) error
	ErrHandler  func(string, error)
)

type Confirmer struct {
	client Client
	queue  *queue.Queue

	confirmationBlocks   uint64
	confirmationInterval int64 // sec
	workers              int
	workerInterval       int64 // milisec
	timeout              int64 // sec

	AfterTxSent      HashHandler
	AfterTxConfirmed HashHandler
	ErrHandler       ErrHandler

	closeCounter uint32
}

type entry struct {
	hash      string
	updatedAt int64
}

func NewConfirmer(client Client, queueSize int, opts ...Opt) Confirmer {
	q := queue.NewQueue(queueSize, false)

	if DEFAULT_WORKERS == 0 {
		DEFAULT_WORKERS = 1
	}

	c := Confirmer{
		client:               client,
		queue:                &q,
		confirmationBlocks:   DEFAULT_CONFIEMATION_BLOCKS,
		confirmationInterval: DEFAULT_CONFIEMATION_INTERVAL,
		workers:              DEFAULT_WORKERS,
		workerInterval:       DEFAULT_WORKER_INTERVAL,
		timeout:              DEFAULT_TIMEOUT,
		AfterTxSent:          DefaultAfterTxSent,
		AfterTxConfirmed:     DefaultAfterTxConfirmed,
		ErrHandler:           DefaultErrHandler,
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

	if err = c.AfterTxSent(hash); err != nil {
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

func (c *Confirmer) DequeueTx(ctx context.Context) (h string, err error) {
	v, isEmpty := c.queue.Dequeue()
	if isEmpty {
		return
	}

	var (
		e   = v.(entry)
		now = time.Now().Unix()
	)

	h = e.hash

	if now < e.updatedAt+c.confirmationInterval {
		if err := c.queue.Enqueue(e); err != nil {
			return e.hash, errors.Wrap(err, "err Enqueue")
		}
		return e.hash, nil
	}

	if err = c.client.ConfirmTx(ctx, e.hash, c.confirmationBlocks); err != nil {
		if errors.Is(err, ErrTxNotFound) || errors.Is(err, ErrTxConfirmPending) {
			e.updatedAt = now
			if err = c.queue.Enqueue(e); err != nil {
				err = errors.Wrap(err, "err Enqueue")
				return
			}
			return
		}

		err = errors.Wrap(err, "err ConfirmTx")
		return
	}

	if err = c.AfterTxConfirmed(e.hash); err != nil {
		err = errors.Wrap(err, "err afterTxSent")
		return
	}

	return
}

func (c *Confirmer) QueueLen() int {
	return c.queue.Len()
}

func (c *Confirmer) Start(ctx context.Context) error {

	worker := func(cancelCtx context.Context, c *Confirmer, id int) {
		timer := time.NewTicker(time.Duration(c.workerInterval) * time.Millisecond)
		defer timer.Stop()

		for {
			select {
			case <-cancelCtx.Done():
				fmt.Printf("worker(%d) is closing\n", id)
				atomic.AddUint32(&c.closeCounter, 1)
				return
			case <-timer.C:
				ctx, cancel := c.withTimeout()
				defer cancel()

				if hash, err := c.DequeueTx(ctx); err != nil {
					c.ErrHandler(hash, err)
				}
			}
		}
	}

	for i := 0; i < c.workers; i++ {
		id := i + 1
		go worker(ctx, c, id)
	}

	fmt.Print("confirmer is ready\n")
	return nil
}

func (c *Confirmer) Close(canncel context.CancelFunc) {
	canncel()
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
	return atomic.LoadUint32(&c.closeCounter) >= uint32(c.workers)
}
