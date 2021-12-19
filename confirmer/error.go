package confirmer

var (
	ErrTxNotFound       = errors.New("tx not found")
	ErrTxFailed         = errors.New("tx failed")
	ErrTxConfirmPending = errors.New("tx confirm pending")
)
