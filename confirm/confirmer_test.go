package confirm

import (
	"context"
	"errors"
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

var MockClientError error

func (c *MockClient) ConfirmTx(ctx context.Context, hash string, confirmationBlocks uint64) error {
	if MockClientError != nil {
		return MockClientError
	}
	if time.Now().Unix()%2 == 0 {
		return ErrTxConfirmPending
	}
	return nil
}

func TestClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := NewConfirmer(&MockClient{}, 0, WithWorkers(2), WithTimeout(3))

	c.Start(ctx)
	c.Close(cancel)
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
		ctx, cancel = context.WithCancel(context.Background())
		txs         = []string{
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
			checker.deleteSent(h)
			return nil
		}
		confirmedCounter = uint32(0)
		confirmed        = func(h string) error {
			checker.deleteConfirmed(h)
			atomic.AddUint32(&confirmedCounter, 1)
			return nil
		}
	)

	c := NewConfirmer(&MockClient{}, 5, WithWorkers(2), WithTimeout(3), WithAfterTxSent(sent), WithAfterTxConfirmed(confirmed))
	err := c.Start(ctx)
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

	c.Close(cancel)

	require.Equal(t, len(checker.unsent), 0)
	require.Equal(t, len(checker.unconfirmed), 0)
}

func TestErrHandle(t *testing.T) {
	var (
		ctx, cancel  = context.WithCancel(context.Background())
		expectedHash = "0x01"
		expectedErr  = ErrTxFailed
		closing      = make(chan struct{})
		errHandler   = func(h string, err error) {
			defer close(closing)

			if !errors.Is(err, expectedErr) {
				panic("unexpected error")
			}

			if h != expectedHash {
				panic("unexpected hash")
			}
		}
	)

	// set error
	MockClientError = ErrTxFailed

	c := NewConfirmer(&MockClient{}, 5, WithWorkers(1), WithErrHandler(errHandler))
	err := c.Start(ctx)
	require.NoError(t, err)

	c.EnqueueTx(context.Background(), expectedHash)

	<-closing

	c.Close(cancel)
	MockClientError = nil
}
