package confirmer

const (
	DEFAULT_CONFIEMATION_BLOCKS   = uint64(2)
	DEFAULT_CONFIEMATION_INTERVAL = uint64(1000) // 1000 ms = 1s
	DEFAULT_WORKERS               = 1
)

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
type ConfirmationInterval uint64

func (i ConfirmationInterval) Apply(c *Confirmer) {
	c.confirmationInterval = uint64(i)
}
func WithConfirmationInterval(i uint64) ConfirmationInterval {
	return ConfirmationInterval(i)
}

// Workers
type Workers int

func (w Workers) Apply(c *Confirmer) {
	c.workers = int(w)
}
func WithWorkers(w int) Workers {
	return Workers(w)
}

// AfterTxSent
type AfterTxSent func(string) error

func (f AfterTxSent) Apply(c *Confirmer) {
	c.afterTxSent = f
}
func WithAfterTxSent(f func(string) error) AfterTxSent {
	return AfterTxSent(f)
}
