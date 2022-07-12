// $ go test -bench=. ./tests/pfm -cpuprofile 4field.prof
// $ go tool pprof -http :6060 4field.prof
//
package types

import (
	"encoding/hex"
	"fmt"
	"github.com/line/lbm-sdk/crypto/keys/secp256k1"
	sdk "github.com/line/lbm-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func BenchmarkBaseAccount(b *testing.B) {
	var err error

	pubkey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.BytesToAccAddress(pubkey.Address())
	account := NewBaseAccount(addr, pubkey, 0, 0)
	pb, err := account.Marshal()
	require.NoError(b, err)
	fmt.Printf("%s (%d bytes)\n", hex.EncodeToString(pb), len(pb))

	b.Run("ProtoBuf/Noop", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			/* */
		}
	})

	b.Run("ProtoBuf/GetPubKey", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = account.GetPubKey()
		}
	})

	b.Run("ProtoBuf/SetPubKey", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err = account.SetPubKey(pubkey)
		}
	})
	require.NoError(b, err)

	b.Run("ProtoBuf/Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err = account.Marshal()
		}
	})
	require.NoError(b, err)

	other := new(BaseAccount)
	b.Run("ProtoBuf/Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err = other.Unmarshal(pb)
		}
	})
	require.NoError(b, err)
}

// % go test ./tests/pfm -cpuprofile 1field.prof -run TestBaseAccountSetPubKey
// % go tool pprof -http :6060 1field.prof
func TestBaseAccountSetPubKey(t *testing.T) {
	var err error
	pubkey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.BytesToAccAddress(pubkey.Address())
	account := NewBaseAccount(addr, pubkey, 0, 0)
	for i := 0; i < 6379531; i++ {
		err = account.SetPubKey(pubkey)
	}
	require.NoError(t, err)
}
