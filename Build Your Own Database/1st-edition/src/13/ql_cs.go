package byodb13

type StmtSplitter struct {
	input []byte
}

func (self *StmtSplitter) Feed(line string) {
	self.input = append(self.input, line...)
}

func (self *StmtSplitter) Pop() ([]byte, bool) {
	idx := 0
loop:
	for idx < len(self.input) {
		switch self.input[idx] {
		case '"', '\'':
			quote := self.input[idx]
			idx++
		str:
			for idx < len(self.input) {
				switch self.input[idx] {
				case quote:
					idx++
					break str
				case '\\':
					idx += 2
				default:
					idx++
				}
			}
		case ';':
			break loop
		default:
			idx++
		}
	}

	if !(idx < len(self.input) && self.input[idx] == ';') {
		return nil, false
	}

	out := self.input[:idx:idx]
	self.input = self.input[idx+1:]
	return out, true
}

func fullStmt(input []byte) (interface{}, error) {
	p := Parser{input: input}
	parsed := pStmt(&p)
	skipSpace(&p)
	if p.idx != len(p.input) {
		pErr(&p, nil, "stmt not ended")
	}
	if p.err != nil {
		return nil, p.err
	}
	return parsed, nil
}

func DBTXExecString(tx *DBTX, input []byte) (QLResult, error) {
	stmt, err := fullStmt(input)
	if err != nil {
		return QLResult{}, err
	}
	return qlExec(tx, &tx.DBReader, stmt)
}

func DBReaderExecString(tx *DBReader, input []byte) (QLResult, error) {
	stmt, err := fullStmt(input)
	if err != nil {
		return QLResult{}, err
	}
	return qlExec(nil, tx, stmt)
}
