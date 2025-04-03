package byodb09

import (
	"fmt"
	"testing"

	is "github.com/stretchr/testify/require"
)

func TestBTreeIter(t *testing.T) {
	{
		c := newC()
		iter := c.tree.SeekLE(nil)
		is.False(t, iter.Valid())
	}

	sizes := []int{5, 2500}
	for _, sz := range sizes {
		c := newC()

		for i := 0; i < sz; i++ {
			key := fmt.Sprintf("key%010d", i)
			val := fmt.Sprintf("vvv%d", fmix32(uint32(-i)))
			c.add(key, val)
		}
		c.verify(t)

		prevk, prevv := []byte(nil), []byte(nil)
		for i := 0; i < sz; i++ {
			key := []byte(fmt.Sprintf("key%010d", i))
			val := []byte(fmt.Sprintf("vvv%d", fmix32(uint32(-i))))
			// fmt.Println(i, string(key), val)

			iter := c.tree.SeekLE(key)
			is.True(t, iter.Valid())
			gotk, gotv := iter.Deref()
			is.Equal(t, key, gotk)
			is.Equal(t, val, gotv)

			iter.Prev()
			if i > 0 {
				is.True(t, iter.Valid())
				gotk, gotv := iter.Deref()
				is.Equal(t, prevk, gotk)
				is.Equal(t, prevv, gotv)
			} else {
				is.False(t, iter.Valid())
			}

			iter.Next()
			{
				is.True(t, iter.Valid())
				gotk, gotv := iter.Deref()
				is.Equal(t, key, gotk)
				is.Equal(t, val, gotv)
			}

			if i+1 == sz {
				iter.Next()
				is.False(t, iter.Valid())
			}

			prevk, prevv = key, val
		}
	}
}
