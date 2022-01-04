package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/tak1827/go-store/store"
	"github.com/tak1827/transaction-confirmer/confirm"
	"github.com/tak1827/transaction-confirmer/sample/log"
	"github.com/tak1827/transaction-confirmer/sample/pb"
)

var (
	Endpoint = "http://localhost:8545"
	PrivKey  = "d1c71e71b06e248c8dbe94d49ef6d6b0d64f5d71b1e33a0f39e14dadb070304a"
)

func main() {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		txStore     = txStore()
	)

	client, err := NewClient(ctx, Endpoint)
	errHandler(err)

	wallet, err := NewWallet(ctx, &client, PrivKey)
	errHandler(err)

	sent := func(hash string) error {
		log.Logger.Info().Msgf("tx sent, hash: %s", hash)
		return nil
	}

	confirmed := func(hash string) error {
		value, err := txStore.Get([]byte(hash))
		var t pb.Transaction
		if err = t.Unmarshal(value); err != nil {
			return err
		}
		log.Logger.Info().Msgf("tx confirmed, tx: %v", t)
		return txStore.Delete([]byte(hash))
	}

	confirmer := confirm.NewConfirmer(&client, 100, confirm.WithWorkers(2), confirm.WithWorkerInterval(100), confirm.WithTimeout(15), confirm.WithAfterTxSent(sent), confirm.WithAfterTxConfirmed(confirmed))

	confirmer.Start(ctx)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	ticker := time.NewTicker(1000 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ch:
			log.Logger.Info().Msg("shutting down...")
			confirmer.Close(cancel)
			return
		case <-ticker.C:
			tx := buildTx(&client, &wallet, txStore)
			err = confirmer.EnqueueTx(ctx, tx)
			errHandler(err)
		default:
		}
	}
}

func errHandler(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func txStore() *store.PrefixStore {
	leveldb, err := store.NewLevelDB("")
	errHandler(err)
	return store.NewPrefixStore(leveldb, pb.PREFIX_PENDING_TX)
}

func buildTx(client *Client, w *Wallet, txStore *store.PrefixStore) *types.Transaction {
	var (
		priv   = w.priv
		nonce  = w.nonce.Increment()
		amount = ToWei(1.0, 9) // 1gwai
	)

	to, err := GenerateAddr()
	errHandler(err)

	tx, err := client.BuildTx(priv, nonce, to, amount)
	errHandler(err)

	now := time.Now()
	transactoin := pb.Transaction{
		Id:        tx.Hash().Hex(),
		To:        to.Hex(),
		Nonce:     tx.Nonce(),
		Status:    pb.Transaction_PENDING,
		UpdatedAt: &now,
	}
	key := transactoin.StoreKey()
	value, err := transactoin.Marshal()
	errHandler(err)

	err = txStore.Put(key, value)
	errHandler(err)

	return tx
}
