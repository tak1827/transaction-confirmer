package confirmer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type MockClient struct{}

func (c *MockClient) SendTx(ctx context.Context, tx interface{}) (string, error) {
	hash := tx.(string)
	return hash, nil
}

func (c *MockClient) ConfirmTx(ctx context.Context, hash string, confirmationBlocks uint64) error {
	if time.Now().Unix()%2 == 0 {
		return ErrTxConfirmPending
	}
	return nil
}

func TestClose(t *testing.T) {
	c := NewConfirmer(&MockClient{}, 100, WithWorkers(2), WithTimeout(3))

	c.Start()
	c.Close()
}

type safeMap struct {
	sync.Mutex
	unsent      map[string]struct{}
	unconfirmed map[string]struct{}
}

func (s *safeMap) deleteSent(hash string) {
	s.Lock()
	defer s.Unlock()

	delete(s.unsent, hash)
}

func (s *safeMap) deleteConfirmed(hash string) {
	s.Lock()
	defer s.Unlock()

	delete(s.unconfirmed, hash)
}

func TestSendTxDequeueTx(t *testing.T) {
	var (
		txs = []string{
			"0x01",
			"0x02",
			"0x03",
			"0x04",
			"0x05",
		}
		checker = safeMap{
			unsent:      make(map[string]struct{}, len(txs)),
			unconfirmed: make(map[string]struct{}, len(txs)),
		}
		sent = func(h string) error {
			fmt.Printf("sent: %s\n", h)
			checker.deleteSent(h)
			return nil
		}
		confirmedCounter = uint32(0)
		confirmed        = func(h string) error {
			fmt.Printf("confirmed: %s\n", h)
			checker.deleteConfirmed(h)
			atomic.AddUint32(&confirmedCounter, 1)
			return nil
		}
	)

	c := NewConfirmer(&MockClient{}, 0, WithWorkers(2), WithTimeout(3), WithAfterTxSent(sent), WithAfterTxConfirmed(confirmed))
	err := c.Start()
	require.NoError(t, err)

	for _, tx := range txs {
		checker.unsent[tx] = struct{}{}
		checker.unconfirmed[tx] = struct{}{}

		c.EnqueueTx(context.Background(), tx)
	}

	for {
		if atomic.LoadUint32(&confirmedCounter) >= uint32(len(txs)) {
			break
		}
	}

	c.Close()

	require.Equal(t, len(checker.unsent), 0)
	require.Equal(t, len(checker.unconfirmed), 0)
}
