package byodb01

import (
	"bufio"
	"os"
)

func LogCreate(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0664)
}

func LogAppend(fp *os.File, line string) error {
	buf := []byte(line)
	buf = append(buf, '\n')
	_, err := fp.Write(buf)
	if err != nil {
		return err
	}
	return fp.Sync() // fsync
}

func LogRead(fp *os.File) ([]string, error) {
	logs := []string{}
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		logs = append(logs, scanner.Text())
	}
	return logs, scanner.Err()
}
