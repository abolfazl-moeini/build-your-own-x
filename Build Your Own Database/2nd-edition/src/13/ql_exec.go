package byodb13

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strconv"
)

// for evaluating expressions
type QLEvalContex struct {
	env Record // optional row values
	out Value
	err error
}

func qlErr(ctx *QLEvalContex, format string, args ...interface{}) {
	if ctx.err == nil {
		ctx.out.Type = QL_ERR
		ctx.err = fmt.Errorf(format, args...)
	}
}

// compare 2 values
func qlValueCmp(ctx *QLEvalContex, a1 Value, a2 Value) int {
	switch {
	case ctx.err != nil:
		return 0
	case a1.Type != a2.Type:
		qlErr(ctx, "comparison of different types")
		return 0
	case a1.Type == TYPE_INT64:
		return cmp.Compare(a1.I64, a2.I64)
	case a1.Type == TYPE_BYTES:
		return bytes.Compare(a1.Str, a2.Str)
	default:
		panic("unreachable")
	}
}

// compare 2 tuples of equal length
func qlTupleCmp(ctx *QLEvalContex, n1 QLNode, n2 QLNode) int {
	if len(n1.Kids) != len(n2.Kids) {
		qlErr(ctx, "tuple comparison of different lengths")
	}
	for i := 0; i < len(n1.Kids) && ctx.err == nil; i++ {
		qlEval(ctx, n1.Kids[i])
		a1 := ctx.out
		qlEval(ctx, n2.Kids[i])
		a2 := ctx.out
		if cmp := qlValueCmp(ctx, a1, a2); cmp != 0 {
			return cmp
		}
	}
	return 0
}

// is the comparison ok?
func cmp2bool(result int, cmd uint32) bool {
	switch cmd {
	case QL_CMP_GE:
		return result >= 0
	case QL_CMP_GT:
		return result > 0
	case QL_CMP_LT:
		return result < 0
	case QL_CMP_LE:
		return result <= 0
	case QL_CMP_EQ:
		return result == 0
	case QL_CMP_NE:
		return result != 0
	default:
		panic("unreachable")
	}
}

// binary operators
func qlBinop(ctx *QLEvalContex, node QLNode) {
	isCmp := false
	switch node.Type {
	case QL_CMP_GE, QL_CMP_GT, QL_CMP_LT, QL_CMP_LE, QL_CMP_EQ, QL_CMP_NE:
		isCmp = true
	}

	// tuple comparison
	if isCmp && node.Kids[0].Type == QL_TUP && node.Kids[1].Type == QL_TUP {
		r := qlTupleCmp(ctx, node.Kids[0], node.Kids[1])
		ctx.out.Type = QL_I64
		ctx.out.I64 = b2i(cmp2bool(r, node.Type))
		return
	}

	// evaluate subexpressions
	qlEval(ctx, node.Kids[0])
	a1 := ctx.out
	// TODO: boolean short circuit
	qlEval(ctx, node.Kids[1])
	a2 := ctx.out

	// scalar comparison
	if isCmp {
		r := qlValueCmp(ctx, a1, a2)
		ctx.out.Type = QL_I64
		ctx.out.I64 = b2i(cmp2bool(r, node.Type))
		return
	}

	switch {
	case ctx.err != nil:
		return
	case a1.Type != a2.Type:
		qlErr(ctx, "binop type mismatch")
	case a1.Type == TYPE_INT64:
		ctx.out.Type = QL_I64
		ctx.out.I64 = qlBinopI64(ctx, node.Type, a1.I64, a2.I64)
	case a1.Type == TYPE_BYTES:
		ctx.out.Type = QL_STR
		qlBinopStr(ctx, node.Type, a1.Str, a2.Str)
	default:
		panic("unreachable")
	}
}

func qlBinopStr(ctx *QLEvalContex, op uint32, a1 []byte, a2 []byte) {
	switch op {
	case QL_ADD:
		ctx.out.Type = TYPE_BYTES
		ctx.out.Str = slices.Concat(a1, a2)
	default:
		qlErr(ctx, "bad str binop")
	}
}

func b2i(b bool) int64 {
	if b {
		return 1
	} else {
		return 0
	}
}

func qlBinopI64(ctx *QLEvalContex, op uint32, a1 int64, a2 int64) int64 {
	switch op {
	case QL_ADD:
		return a1 + a2
	case QL_SUB:
		return a1 - a2
	case QL_MUL:
		return a1 * a2
	case QL_DIV:
		if a2 == 0 {
			qlErr(ctx, "division by zero")
			return 0
		}
		return a1 / a2
	case QL_MOD:
		if a2 == 0 {
			qlErr(ctx, "division by zero")
			return 0
		}
		return a1 % a2
	case QL_AND:
		return b2i(a1&a2 != 0)
	case QL_OR:
		return b2i(a1|a2 != 0)
	default:
		qlErr(ctx, "bad i64 binop")
		return 0
	}
}

// evaluate an expression recursively
func qlEval(ctx *QLEvalContex, node QLNode) {
	if ctx.err != nil {
		return
	}

	switch node.Type {
	// refer to a column
	case QL_SYM:
		if v := ctx.env.Get(string(node.Str)); v != nil {
			ctx.out = *v
		} else {
			qlErr(ctx, "unknown column: %s", node.Str)
		}
	// a literal value
	case QL_I64, QL_STR:
		ctx.out = node.Value
	case QL_TUP:
		qlErr(ctx, "unexpected tuple")
	// unary ops
	case QL_NEG:
		qlEval(ctx, node.Kids[0])
		if ctx.out.Type == TYPE_INT64 {
			ctx.out.I64 = -ctx.out.I64
		} else {
			qlErr(ctx, "QL_NEG type error")
		}
	case QL_NOT:
		qlEval(ctx, node.Kids[0])
		if ctx.out.Type == TYPE_INT64 {
			ctx.out.I64 = b2i(ctx.out.I64 == 0)
		} else {
			qlErr(ctx, "QL_NOT type error")
		}
	// binary ops
	case QL_CMP_GE, QL_CMP_GT, QL_CMP_LT, QL_CMP_LE, QL_CMP_EQ, QL_CMP_NE:
		fallthrough
	case QL_ADD, QL_SUB, QL_MUL, QL_DIV, QL_MOD, QL_AND, QL_OR:
		qlBinop(ctx, node)
	default:
		panic("unreachable")
	}
}

// convert `INDEX BY` to `Record` and `CMP_??`
func qlEvalScanKey(node QLNode) (Record, int, error) {
	cmp := 0
	switch node.Type {
	case QL_CMP_GE:
		cmp = CMP_GE
	case QL_CMP_GT:
		cmp = CMP_GT
	case QL_CMP_LT:
		cmp = CMP_LT
	case QL_CMP_LE:
		cmp = CMP_LE
	case QL_CMP_EQ:
		cmp = 0 // later
	case 0:
		return Record{}, 0, nil // non-existent; handled later
	default:
		panic("unreachable")
	}
	names, exprs := node.Kids[0], node.Kids[1]
	assert(names.Type == QL_TUP && exprs.Type == QL_TUP)
	assert(len(names.Kids) == len(exprs.Kids))

	vals, err := qlEvelMulti(Record{}, exprs.Kids)
	if err != nil {
		return Record{}, 0, err
	}
	cols := []string{}
	for i := range names.Kids {
		assert(names.Kids[i].Type == QL_SYM)
		cols = append(cols, string(names.Kids[i].Str))
	}
	return Record{cols, vals}, cmp, nil
}

// create the `Scanner` from the `INDEX BY` clause
func qlScanInit(req *QLScan, sc *Scanner) (err error) {
	// convert `QLNode` to `Record` and `CMP_??`
	if sc.Key1, sc.Cmp1, err = qlEvalScanKey(req.Key1); err != nil {
		return err
	}
	if sc.Key2, sc.Cmp2, err = qlEvalScanKey(req.Key2); err != nil {
		return err
	}
	// convert keys to a range
	switch {
	case req.Key1.Type == 0 && req.Key2.Type == 0:
		// no `INDEX BY`; full table scan by the primary key
		sc.Cmp1, sc.Cmp2 = CMP_GE, CMP_LE // compare with infinity
	case req.Key1.Type == QL_CMP_EQ && req.Key2.Type == 0:
		// equal by a prefix: INDEX BY key = val
		sc.Key2 = sc.Key1
		sc.Cmp1, sc.Cmp2 = CMP_GE, CMP_LE
	case req.Key1.Type != 0 && req.Key2.Type == 0:
		// open-ended range: INDEX BY key > val
		if sc.Cmp1 > 0 {
			sc.Cmp2 = CMP_LE // compare with a zero-length tuple
		} else {
			sc.Cmp2 = CMP_GE
		}
	case req.Key1.Type != 0 && req.Key2.Type != 0:
		// two-side range: INDEX BY val1 < key AND key < val2
	default:
		panic("unreachable")
	}
	return nil
}

type RecordIter interface {
	Valid() bool
	Next()
	Deref(*Record) error
}

type qlScanIter struct {
	// input
	req *QLScan
	sc  Scanner
	// state
	idx int64
	end bool
	// cached output item
	rec Record
	err error
}

func qlScanPull(iter *qlScanIter, rec *Record) (bool, error) {
	if iter.idx < iter.req.Offset {
		return false, nil
	}
	iter.sc.Deref(rec)
	if iter.req.Filter.Type != 0 {
		ctx := QLEvalContex{env: *rec}
		qlEval(&ctx, iter.req.Filter)
		if ctx.err != nil {
			return false, ctx.err
		}
		if ctx.out.Type != TYPE_INT64 {
			return false, errors.New("filter is not of boolean type")
		}
		if ctx.out.I64 == 0 {
			return false, nil
		}
	}
	return true, nil
}

func (iter *qlScanIter) Next() {
	for iter.idx < iter.req.Limit && iter.sc.Valid() {
		// check the current item
		got, err := qlScanPull(iter, &iter.rec)
		iter.err = err
		// move to the next one
		iter.idx++
		iter.sc.Next()
		// return if the current item yields result
		if got || err != nil {
			return
		}
	}
	iter.end = true
}

func (iter *qlScanIter) Valid() bool {
	return !iter.end
}

func (iter *qlScanIter) Deref(rec *Record) error {
	assert(iter.Valid())
	if iter.err == nil {
		*rec = iter.rec
	}
	return iter.err
}

// execute a query
func qlScan(req *QLScan, tx *DBTX) (RecordIter, error) {
	iter := qlScanIter{req: req}
	if err := qlScanInit(req, &iter.sc); err != nil {
		return nil, err
	}
	if err := tx.Scan(req.Table, &iter.sc); err != nil {
		return nil, err
	}
	iter.Next()
	return &iter, nil
}

type qlSelectIter struct {
	iter  RecordIter
	names []string
	exprs []QLNode
}

func (iter *qlSelectIter) Valid() bool {
	return iter.iter.Valid()
}
func (iter *qlSelectIter) Next() {
	iter.iter.Next()
}
func (iter *qlSelectIter) Deref(rec *Record) error {
	if err := iter.iter.Deref(rec); err != nil {
		return err
	}
	vals, err := qlEvelMulti(*rec, iter.exprs)
	if err != nil {
		return err
	}
	*rec = Record{iter.names, vals}
	return nil
}

// stmt: select
func qlSelect(req *QLSelect, tx *DBTX) (RecordIter, error) {
	// records
	records, err := qlScan(&req.QLScan, tx)
	if err != nil {
		return nil, err
	}

	// expand the `*`` into columns
	tdef := getTableDef(tx, req.Table)
	names, exprs := []string{}, []QLNode{}
	for i := range req.Names {
		if req.Names[i] != "*" {
			names = append(names, req.Names[i])
			exprs = append(exprs, req.Output[i])
		} else {
			names = append(names, tdef.Cols...)
			for _, col := range tdef.Cols {
				node := QLNode{Value: Value{Type: QL_SYM, Str: []byte(col)}}
				exprs = append(exprs, node)
			}
		}
	}
	assert(len(names) == len(exprs))

	// assign column names if missing
	for i := range names {
		if names[i] != "" {
			continue
		}
		if exprs[i].Type == QL_SYM {
			names[i] = string(exprs[i].Str)
		} else {
			names[i] = strconv.Itoa(i)
		}
	}

	// output
	return &qlSelectIter{iter: records, names: names, exprs: exprs}, nil
}

// stmt: create table
func qlCreateTable(req *QLCreateTable, tx *DBTX) error {
	return tx.TableNew(&req.Def)
}

// stmt: insert
func qlInsert(req *QLInsert, tx *DBTX) (uint64, uint64, error) {
	added, updated := uint64(0), uint64(0)
	for _, nodes := range req.Values {
		// verify values
		vals, err := qlEvelMulti(Record{}, nodes)
		if err != nil {
			return 0, 0, err
		}
		// perform updates
		dbreq := DBUpdateReq{Record: Record{req.Names, vals}, Mode: req.Mode}
		_, err = tx.Set(req.Table, &dbreq)
		if err != nil {
			return 0, 0, err
		}
		// stats
		if dbreq.Added {
			added++
		}
		if dbreq.Updated {
			updated++
		}
	}
	return added, updated, nil
}

// stmt: delete
func qlDelete(req *QLDelete, tx *DBTX) (uint64, error) {
	records, err := qlScan(&req.QLScan, tx)
	if err != nil {
		return 0, err
	}

	tdef := getTableDef(tx, req.Table)
	deleted := uint64(0)
	for ; records.Valid(); records.Next() {
		rec := Record{}
		if err := records.Deref(&rec); err != nil {
			return 0, err
		}
		deleted++

		vals, err := getValues(tdef, rec, tdef.Indexes[0])
		assert(err == nil)
		deleted, err := tx.Delete(req.Table, Record{tdef.Indexes[0], vals})
		assert(err == nil && deleted) // deleting an existing row
	}
	return deleted, nil
}

// stmt: update
func qlUpdate(req *QLUpdate, tx *DBTX) (uint64, error) {
	// no update to the primary key
	assert(len(req.Names) == len(req.Values))
	tdef := getTableDef(tx, req.Table)
	for _, col := range req.Names {
		if slices.Index(tdef.Cols, col) < 0 {
			return 0, fmt.Errorf("unknown column: %s", col)
		}
		if slices.Index(tdef.Indexes[0], col) >= 0 {
			return 0, errors.New("cannot update the primary key")
		}
	}

	// old records
	records, err := qlScan(&req.QLScan, tx)
	if err != nil {
		return 0, err
	}

	updated := uint64(0)
	for ; records.Valid(); records.Next() {
		// old record
		rec := Record{}
		if err := records.Deref(&rec); err != nil {
			return 0, err
		}
		// new record
		vals, err := qlEvelMulti(rec, req.Values)
		if err != nil {
			return 0, err
		}
		for i, col := range req.Names {
			rec.Vals[slices.Index(tdef.Cols, col)] = vals[i]
		}
		// perform updates
		dbreq := DBUpdateReq{Record: rec, Mode: MODE_UPDATE_ONLY}
		if _, err := tx.Set(req.Table, &dbreq); err != nil {
			return 0, err
		}
		// stats
		if dbreq.Updated {
			updated++
		}
	}
	return updated, nil
}

func qlEvelMulti(env Record, exprs []QLNode) ([]Value, error) {
	vals := []Value{}
	for _, node := range exprs {
		ctx := QLEvalContex{env: env}
		qlEval(&ctx, node)
		if ctx.err != nil {
			return nil, ctx.err
		}
		vals = append(vals, ctx.out)
	}
	return vals, nil
}

type QLResult struct {
	Records RecordIter
	Added   uint64
	Updated uint64
	Deleted uint64
}

// execute a single statement
func qlExec(tx *DBTX, stmt interface{}) (result QLResult, err error) {
	save := TXSave{}
	tx.Save(&save)
	switch req := stmt.(type) {
	case *QLSelect:
		result.Records, err = qlSelect(req, tx)
	case *QLCreateTable:
		err = qlCreateTable(req, tx)
	case *QLInsert:
		result.Added, result.Updated, err = qlInsert(req, tx)
	case *QLDelete:
		result.Deleted, err = qlDelete(req, tx)
	case *QLUpdate:
		result.Updated, err = qlUpdate(req, tx)
	default:
		panic("unreachable")
	}
	if err != nil {
		tx.Revert(&save) // revert any partial updates
	}
	return
}
