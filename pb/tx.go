package pb

var PREFIX_PENDING_TX = []byte(".pendingtx")

func (x *Transaction) StoreKey() []byte {
	return []byte(x.GetId())
}
