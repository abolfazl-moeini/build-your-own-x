package byodb01

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
)

func SaveData1(path string, data []byte) error {
	fp, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	if err != nil {
		return err
	}
	defer fp.Close()

	_, err = fp.Write(data)
	if err != nil {
		return err
	}
	return fp.Sync() // fsync
}

func randomInt() int {
	var buf [8]byte
	rand.Read(buf[:])
	return int(binary.LittleEndian.Uint64(buf[:]))
}

func SaveData2(path string, data []byte) error {
	tmp := fmt.Sprintf("%s.tmp.%d", path, randomInt())
	fp, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0664)
	if err != nil {
		return err
	}
	defer func() { // 4. discard the temporary file if it still exists
		fp.Close() // not expected to fail
		if err != nil {
			os.Remove(tmp)
		}
	}()

	if _, err = fp.Write(data); err != nil { // 1. save to the temporary file
		return err
	}
	if err = fp.Sync(); err != nil { // 2. fsync
		return err
	}
	err = os.Rename(tmp, path) // 3. replace the target
	return err
}
