package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	// "github.com/davecgh/go-spew/spew"
	"github.com/tak1827/go-store/store"
	"github.com/tak1827/transaction-confirmer/pb"
)

var (
	// geth
	Endpoint = "http://127.0.0.1:8545"
	PrivKey  = "d1c71e71b06e248c8dbe94d49ef6d6b0d64f5d71b1e33a0f39e14dadb070304a"
	PrivKey2 = "8179ce3d00ac1d1d1d38e4f038de00ccd0e0375517164ac5448e3acc847acb34"
	PrivKey3 = "df38daebd09f56398cc8fd699b72f5ea6e416878312e1692476950f427928e7d"
)

func errHandler(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func main() {
	leveldb, err := store.NewLevelDB("")
	errHandler(err)
	txStore := store.NewPrefixStore(leveldb, pb.PREFIX_PENDING_TX)

	ctx := context.Background()
	client, err := newClient(Endpoint)
	errHandler(err)

	priv, err := crypto.HexToECDSA(PrivKey)
	errHandler(err)

	nonce, err := client.Nonce(ctx, PrivKey)
	errHandler(err)

	to, err := GenerateAddr()
	errHandler(err)

	amount := ToWei(1.0, 9) // 1gwai

	tx, err := client.BuildTx(priv, nonce, to, amount)
	errHandler(err)

	hash, err := client.SendTx(ctx, tx)
	errHandler(err)

	now := time.Now()
	transactoin := pb.Transaction{
		Id:        hash,
		To:        to.String(),
		Nonce:     nonce,
		Status:    pb.Transaction_PENDING,
		UpdatedAt: &now,
	}
	key := transactoin.StoreKey()
	value, err := transactoin.Marshal()
	errHandler(err)
	err = txStore.Put(key, value)
	errHandler(err)

	confirmed := false
	counter := 0
	for !confirmed {
		counter += 1
		fmt.Printf("count: %d\n", counter)
		err = client.ConfirmTx(ctx, hash, 0)
		if err == nil {
			confirmed = true

			value, err = txStore.Get(key)
			errHandler(err)
			var t pb.Transaction
			err = t.Unmarshal(value)
			errHandler(err)
			fmt.Printf("t: %v\n", t)
			err = txStore.Delete(key)
			errHandler(err)
			continue
		}

		if errors.Is(err, ErrTxNotFound) || errors.Is(err, ErrTxConfirmPending) {
			time.Sleep(1)
			continue
		}

		errHandler(err)
	}
}
