package confirm

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/lithdew/bytesutil"
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

	e := &queue.Entry{
		Key:   hash,
		Value: bytesutil.AppendUint64LE([]byte{}, uint64(time.Now().Unix())),
	}

	if err = c.queue.Enqueue(e); err != nil {
		return errors.Wrap(err, "err Enqueue")
	}

	return nil
}

func (c *Confirmer) DequeueTx(ctx context.Context) (string, error) {
	e, isEmpty := c.queue.Dequeue()
	if isEmpty {
		return "", nil
	}

	var (
		hash      = e.Key
		updatedAt = int64(bytesutil.Uint64LE(e.Value))
		now       = time.Now().Unix()
	)

	if now < updatedAt+c.confirmationInterval {
		if err := c.queue.Enqueue(e); err != nil {
			return hash, errors.Wrap(err, "err Enqueue")
		}
		return hash, nil
	}

	if err := c.client.ConfirmTx(ctx, hash, c.confirmationBlocks); err != nil {
		if errors.Is(err, ErrTxNotFound) || errors.Is(err, ErrTxConfirmPending) {
			e.Value = bytesutil.AppendUint64LE([]byte{}, uint64(now))
			if err = c.queue.Enqueue(e); err != nil {
				return hash, errors.Wrap(err, "err Enqueue")
			}
			return hash, nil
		}

		return hash, errors.Wrap(err, "err ConfirmTx")
	}

	if err := c.AfterTxConfirmed(hash); err != nil {
		return hash, errors.Wrap(err, "err afterTxSent")
	}

	return hash, nil
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
