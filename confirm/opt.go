package confirm

import (
	"runtime"
)

const (
	DEFAULT_CONFIEMATION_BLOCKS   = uint64(2)
	DEFAULT_CONFIEMATION_INTERVAL = int64(1) // 1s
	DEFAULT_WORKER_INTERVAL       = int64(10)
	DEFAULT_TIMEOUT               = int64(60)
)

var (
	DEFAULT_WORKERS = runtime.NumCPU() - 2
)

func DefaultAfterTxSent(hash string) error {
	return nil
}

func DefaultAfterTxConfirmed(hash string) error {
	return nil
}

func DefaultErrHandler(hash string, err error) {
	panic(err.Error())
}

type Opt interface {
	Apply(c *Confirmer)
}

// ConfirmationBlocks
type ConfirmationBlocks uint64

func (b ConfirmationBlocks) Apply(c *Confirmer) {
	c.confirmationBlocks = uint64(b)
}
func WithConfirmationBlock(b uint64) ConfirmationBlocks {
	return ConfirmationBlocks(b)
}

// ConfirmationInterval
type ConfirmationInterval int64

func (i ConfirmationInterval) Apply(c *Confirmer) {
	c.confirmationInterval = int64(i)
}
func WithConfirmationInterval(i int64) ConfirmationInterval {
	return ConfirmationInterval(i)
}

// WorkerInterval
type WorkerInterval int64

func (i WorkerInterval) Apply(c *Confirmer) {
	c.workerInterval = int64(i)
}
func WithWorkerInterval(i int64) WorkerInterval {
	return WorkerInterval(i)
}

// Workers
type Workers int

func (w Workers) Apply(c *Confirmer) {
	c.workers = int(w)
}
func WithWorkers(w int) Workers {
	if w <= 0 {
		panic("workers should be positive")
	}
	return Workers(w)
}

// Timeout
type Timeout int64

func (t Timeout) Apply(c *Confirmer) {
	c.timeout = int64(t)
}
func WithTimeout(t int64) Timeout {
	if t <= 0 {
		panic("Timeout should be positive")
	}
	return Timeout(t)
}

// AfterTxSent
type AfterTxSent func(string) error

func (f AfterTxSent) Apply(c *Confirmer) {
	c.AfterTxSent = HashHandler(f)
}
func WithAfterTxSent(f func(string) error) AfterTxSent {
	return AfterTxSent(f)
}

// AfterTxConfirmed
type AfterTxConfirmed func(string) error

func (f AfterTxConfirmed) Apply(c *Confirmer) {
	c.AfterTxConfirmed = HashHandler(f)
}
func WithAfterTxConfirmed(f func(string) error) AfterTxConfirmed {
	return AfterTxConfirmed(f)
}

func (f ErrHandler) Apply(c *Confirmer) {
	c.ErrHandler = f
}
func WithErrHandler(f func(string, error)) ErrHandler {
	return ErrHandler(f)
}
