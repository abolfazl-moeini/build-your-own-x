package byodb11

import (
	"testing"

	is "github.com/stretchr/testify/require"
)

func TestKVTX(t *testing.T) {
	d := newD()

	d.add("k1", "v1")

	tx := KVTX{}
	d.db.Begin(&tx)

	tx.Update(&InsertReq{Key: []byte("k1"), Val: []byte("xxx")})
	tx.Update(&InsertReq{Key: []byte("k2"), Val: []byte("xxx")})

	val, ok := tx.Get([]byte("k1"))
	is.True(t, ok)
	is.Equal(t, []byte("xxx"), val)
	val, ok = tx.Get([]byte("k2"))
	is.Equal(t, []byte("xxx"), val)

	d.db.Abort(&tx)

	d.verify(t)

	d.reopen()
	d.verify(t)
	{
		tx := KVTX{}
		d.db.Begin(&tx)
		val, ok = tx.Get([]byte("k2"))
		is.False(t, ok)
		d.db.Abort(&tx)
	}

	d.dispose()
}
