package confirm

import (
	"errors"
)

var (
	ErrTxNotFound                 = errors.New("tx not found")
	ErrTxFailed                   = errors.New("tx failed")
	ErrTxConfirmPending           = errors.New("tx confirm pending")
	ErrQueueIsEmpty               = errors.New("queue is empty")
	ErrBeforeConfirmationInterval = errors.New("before confirmation interval")
)
