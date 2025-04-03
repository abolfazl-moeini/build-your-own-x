package byodb13

import (
	"fmt"
	"slices"
	"sort"
	"testing"

	is "github.com/stretchr/testify/require"
)

func TestKVTXSequential(t *testing.T) {
	d := newD()

	d.add("k1", "v1")

	tx := KVTX{}
	d.db.Begin(&tx)

	tx.Update(&UpdateReq{Key: []byte("k1"), Val: []byte("xxx")})
	tx.Update(&UpdateReq{Key: []byte("k2"), Val: []byte("xxx")})

	val, ok := tx.Get([]byte("k1"))
	is.True(t, ok)
	is.Equal(t, []byte("xxx"), val)
	val, ok = tx.Get([]byte("k2"))
	is.True(t, ok)
	is.Equal(t, []byte("xxx"), val)

	d.db.Abort(&tx)

	d.verify(t)

	d.reopen()
	d.verify(t)
	{
		tx := KVTX{}
		d.db.Begin(&tx)
		_, ok = tx.Get([]byte("k2"))
		is.False(t, ok)
		d.db.Abort(&tx)
	}

	d.dispose()
}

func TestKVTXInterleave(t *testing.T) {
	d := newD()

	{
		tx1, tx2 := KVTX{}, KVTX{}
		d.db.Begin(&tx1)
		d.db.Begin(&tx2)
		tx1.Set([]byte("k1"), []byte("v1"))
		tx2.Set([]byte("k2"), []byte("v2"))

		val, ok := tx1.Get([]byte("k1"))
		assert(ok && string(val) == "v1") // read uncomitted write
		val, ok = tx2.Get([]byte("k2"))
		assert(ok && string(val) == "v2")

		err := d.db.Commit(&tx1)
		assert(err == nil)

		err = d.db.Commit(&tx2)
		assert(err == nil)
		assert(len(d.db.ongoing)+len(d.db.history) == 0)
	}

	{
		tx1, tx2 := KVTX{}, KVTX{}
		d.db.Begin(&tx1)
		d.db.Begin(&tx2)
		tx1.Set([]byte("k1"), []byte("v2"))
		err := d.db.Commit(&tx1)
		assert(err == nil)

		val, ok := tx2.Get([]byte("k1"))
		assert(ok && string(val) == "v1") // isolation

		err = d.db.Commit(&tx2)
		assert(err == nil) // read-only
		assert(len(d.db.ongoing)+len(d.db.history) == 0)
	}

	{
		tx1, tx2 := KVTX{}, KVTX{}
		d.db.Begin(&tx1)
		d.db.Begin(&tx2)
		tx1.Set([]byte("k1"), []byte("v3"))
		err := d.db.Commit(&tx1)
		assert(err == nil)

		val, ok := tx2.Get([]byte("k1"))
		assert(ok && string(val) == "v2")
		tx2.Set([]byte("k2"), val)

		err = d.db.Commit(&tx2)
		assert(err == ErrorConflict) // read conflict
		assert(len(d.db.ongoing)+len(d.db.history) == 0)
	}

	{
		tx1, tx2 := KVTX{}, KVTX{}
		d.db.Begin(&tx1)
		d.db.Begin(&tx2)
		tx1.Set([]byte("k3"), []byte("v1"))
		tx2.Del(&DeleteReq{Key: []byte("k3")})
		err := d.db.Commit(&tx1)
		assert(err == nil)
		err = d.db.Commit(&tx2)
		assert(err == ErrorConflict) // write conflict
		assert(len(d.db.ongoing)+len(d.db.history) == 0)
	}

	{
		d.add("k4", "v1")
		tx1, tx2 := KVTX{}, KVTX{}
		d.db.Begin(&tx1)
		d.db.Begin(&tx2)
		tx1.Set([]byte("k4"), []byte("v2"))
		tx2.Del(&DeleteReq{Key: []byte("k4")})
		err := d.db.Commit(&tx2)
		assert(err == nil)
		err = d.db.Commit(&tx1)
		assert(err == ErrorConflict) // write conflict
		assert(len(d.db.ongoing)+len(d.db.history) == 0)
	}

	{
		tx1, tx2 := KVTX{}, KVTX{}
		d.db.Begin(&tx1)
		d.db.Begin(&tx2)
		tx1.Set([]byte("k5"), []byte("v2"))
		tx2.Del(&DeleteReq{Key: []byte("k5")})
		err := d.db.Commit(&tx2) // no write
		assert(err == nil)
		err = d.db.Commit(&tx1)
		assert(err == nil) // no conflict
		assert(len(d.db.ongoing)+len(d.db.history) == 0)
	}

	{
		tx1, tx2 := KVTX{}, KVTX{}
		d.db.Begin(&tx1)
		d.add("k6", "v1") // 3rd TX
		d.add("k7", "v1")

		d.db.Begin(&tx2)
		tx2.Set([]byte("k6"), []byte("v2"))
		err := d.db.Commit(&tx2) // no conflict
		assert(err == nil)

		_, ok := tx1.Get([]byte("k7"))
		assert(!ok)
		tx1.Set([]byte("k8"), []byte("v3"))
		err = d.db.Commit(&tx1) // read conflict
		assert(err == ErrorConflict)
		assert(len(d.db.ongoing)+len(d.db.history) == 0)
	}

	d.dispose()
}

func TestKVTXRand(t *testing.T) {
	d := newD()
	order := []uint32{}
	funcs := []func(){}

	N := uint32(50_000)
	for i := uint32(0); i < N; i++ {
		tx := KVTX{}
		key, val := fmt.Sprintf("k%v", i), fmt.Sprintf("v%v", i)
		funcs = append(funcs, func() { d.db.Begin(&tx) })
		funcs = append(funcs, func() { tx.Set([]byte(key), []byte(val)) })
		funcs = append(funcs, func() {
			err := d.db.Commit(&tx)
			assert(err == nil)
		})

		nums := []uint32{fmix32(3*i + 0), fmix32(3*i + 1), fmix32(3*i + 2)}
		slices.Sort(nums)
		order = append(order, nums...)
	}
	sort.Sort(sortIF{
		len:  int(N),
		less: func(i, j int) bool { return order[i] < order[j] },
		swap: func(i, j int) {
			order[i], order[j] = order[j], order[i]
			funcs[i], funcs[j] = funcs[j], funcs[i]
		},
	})

	for _, f := range funcs {
		f()
	}
	assert(len(d.db.ongoing)+len(d.db.history) == 0)

	d.dispose()
}
