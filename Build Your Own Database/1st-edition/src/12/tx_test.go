package byodb12

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

func TestKVRW(t *testing.T) {
	d := newD()

	{
		r0 := KVReader{}
		d.db.BeginRead(&r0)

		d.add("k1", "v1")
		{
			r1 := KVReader{}
			d.db.BeginRead(&r1)

			d.add("k2", "v2")
			val, ok := r1.Get([]byte("k2"))
			assert(!ok)
			val, ok = r1.Get([]byte("k1"))
			assert(ok && string(val) == "v1")

			d.db.EndRead(&r1)
		}

		d.add("k3", "v3")

		_, ok := r0.Get([]byte("k1"))
		assert(!ok)
		_, ok = r0.Get([]byte("k2"))
		assert(!ok)
		_, ok = r0.Get([]byte("k3"))
		assert(!ok)

		d.db.EndRead(&r0)
	}

	{
		r3 := KVReader{}
		d.db.BeginRead(&r3)
		val, ok := r3.Get([]byte("k1"))
		assert(ok && string(val) == "v1")
		val, ok = r3.Get([]byte("k2"))
		assert(ok && string(val) == "v2")
		val, ok = r3.Get([]byte("k3"))
		assert(ok && string(val) == "v3")
		d.db.EndRead(&r3)
	}

	assert(d.db.version == 3)

	d.dispose()
}
