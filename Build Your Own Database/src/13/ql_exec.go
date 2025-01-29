package byodb13

import (
	"bytes"
	"errors"
	"fmt"
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
	if a1.Type != a2.Type {
		qlErr(ctx, "comparison of different types")
	}
	if ctx.err != nil {
		return 0
	}

	switch a1.Type {
	case TYPE_INT64:
		switch {
		case a1.I64 < a2.I64:
			return -1
		case a1.I64 > a2.I64:
			return +1
		default:
			return 0
		}
	case TYPE_BYTES:
		return bytes.Compare(a1.Str, a2.Str)
	default:
		panic("not implemented")
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
		cmp := qlValueCmp(ctx, a1, a2)
		if cmp != 0 {
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
		ctx.out.I64 = b2i64(cmp2bool(r, node.Type))
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
		ctx.out.I64 = b2i64(cmp2bool(r, node.Type))
		return
	}

	if a1.Type != a2.Type {
		qlErr(ctx, "binop type mismatch")
	}
	if ctx.err != nil {
		return
	}

	switch a1.Type {
	case TYPE_INT64:
		ctx.out.Type = QL_I64
		ctx.out.I64 = qlBinopI64(ctx, node.Type, a1.I64, a2.I64)
	case TYPE_BYTES:
		ctx.out.Type = QL_STR
		qlBinopStr(ctx, node.Type, a1.Str, a2.Str)
	default:
		panic("not implemented")
	}
}

func qlBinopStr(ctx *QLEvalContex, op uint32, a1 []byte, a2 []byte) {
	switch op {
	case QL_ADD:
		ctx.out.Type = TYPE_BYTES
		ctx.out.Str = append(a1[:len(a1):len(a1)], a2...)
	default:
		qlErr(ctx, "bad str binop")
	}
}

func b2i64(b bool) int64 {
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
		return b2i64(a1&a2 != 0)
	case QL_OR:
		return b2i64(a1|a2 != 0)
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
			ctx.out.I64 = b2i64(ctx.out.I64 == 0)
		} else {
			qlErr(ctx, "QL_NOT type error")
		}
	// binary ops
	case QL_CMP_GE, QL_CMP_GT, QL_CMP_LT, QL_CMP_LE, QL_CMP_EQ, QL_CMP_NE:
		fallthrough
	case QL_ADD, QL_SUB, QL_MUL, QL_DIV, QL_MOD, QL_AND, QL_OR:
		qlBinop(ctx, node)
	default:
		panic("not implemented")
	}
}

// check the `INDEX BY` clause
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
	default:
		panic("unreachable")
	}

	names, exprs := node.Kids[0], node.Kids[1]
	assert(names.Type == QL_TUP && exprs.Type == QL_TUP)
	assert(len(names.Kids) == len(exprs.Kids))

	rec := Record{}
	for i := range names.Kids {
		assert(names.Kids[i].Type == QL_SYM)
		ctx := QLEvalContex{}
		qlEval(&ctx, exprs.Kids[i])
		if ctx.err != nil {
			return Record{}, 0, ctx.err
		}

		rec.Cols = append(rec.Cols, string(names.Kids[i].Str))
		rec.Vals = append(rec.Vals, ctx.out)
	}

	return rec, cmp, nil
}

// create the `Scanner` from the `INDEX BY` clause
func qlScanInit(req *QLScan, sc *Scanner) error {
	if req.Key1.Type == 0 {
		// no `INDEX BY`; scan by the primary key
		sc.Cmp1, sc.Cmp2 = CMP_GE, CMP_LE
		return nil
	}

	var err error
	sc.Key1, sc.Cmp1, err = qlEvalScanKey(req.Key1)
	if err != nil {
		return err
	}

	if req.Key2.Type != 0 {
		sc.Key2, sc.Cmp2, err = qlEvalScanKey(req.Key1)
		if err != nil {
			return err
		}
	}

	if req.Key1.Type == QL_CMP_EQ && req.Key2.Type != 0 {
		return errors.New("bad `INDEX BY`")
	}
	if req.Key1.Type == QL_CMP_EQ {
		sc.Key2 = sc.Key1
		sc.Cmp1, sc.Cmp2 = CMP_GE, CMP_LE
	}
	return nil
}

// fetch all rows from a `Scanner`
func qlScanRun(req *QLScan, sc *Scanner, out []Record) ([]Record, error) {
	for i := int64(0); sc.Valid(); i++ {
		ok := req.Offset <= i && i < req.Limit

		rec := Record{}
		if ok {
			sc.Deref(&rec)
		}

		if ok && req.Filter.Type != 0 {
			ctx := QLEvalContex{env: rec}
			qlEval(&ctx, req.Filter)
			if ctx.err != nil {
				return nil, ctx.err
			}
			if ctx.out.Type != TYPE_INT64 {
				return nil, errors.New("filter is not of boolean type")
			}
			ok = (ctx.out.I64 != 0)
		}

		if ok {
			out = append(out, rec)
		}
		sc.Next()
	}
	return out, nil
}

// execute a query
func qlScan(req *QLScan, tx *DBReader, out []Record) ([]Record, error) {
	sc := Scanner{}
	err := qlScanInit(req, &sc)
	if err != nil {
		return nil, err
	}

	err = tx.Scan(req.Table, &sc)
	if err != nil {
		return nil, err
	}

	return qlScanRun(req, &sc, out)
}

// stmt: select
func qlSelect(req *QLSelect, tx *DBReader, out []Record) ([]Record, error) {
	// records
	records, err := qlScan(&req.QLScan, tx, out)
	if err != nil {
		return nil, err
	}

	// expand the `*`` into columns
	names, exprs := qlExpandStar(req, tx)
	assert(len(names) == len(exprs))

	// assign column names if missing
	names = append([]string(nil), names...)
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
	for _, irec := range records {
		orec := Record{Cols: names}
		for _, node := range exprs {
			ctx := QLEvalContex{env: irec}
			qlEval(&ctx, node)
			if ctx.err != nil {
				return nil, ctx.err
			}
			orec.Vals = append(orec.Vals, ctx.out)
		}
		out = append(out, orec)
	}
	return out, nil
}

func qlExpandStar(req *QLSelect, tx *DBReader) ([]string, []QLNode) {
	// columns to be inserted into stars
	var starExprs []QLNode
	var starNames []string
	for _, node := range req.Output {
		if node.Type != QL_STAR {
			continue
		}
		tdef := getTableDef(tx, req.Table)
		starNames = tdef.Cols
		for _, col := range tdef.Cols {
			starExprs = append(starExprs,
				QLNode{Value: Value{Type: QL_SYM, Str: []byte(col)}})
		}
		break
	}

	// replace stars
	names := req.Names
	exprs := req.Output
	for i := len(exprs) - 1; i >= 0; i-- {
		if exprs[i].Type != QL_STAR {
			continue
		}
		assert(names[i] == "*")
		names = append(append(names[:i:i], starNames...), names[i+1:]...)
		exprs = append(append(exprs[:i:i], starExprs...), exprs[i+1:]...)
	}
	return names, exprs
}

// stmt: create table
func qlCreateTable(req *QLCreateTable, tx *DBTX) error {
	return tx.TableNew(&req.Def)
}

// stmt: insert
func qlInsert(req *QLInsert, tx *DBTX) (uint64, uint64, error) {
	added, updated := uint64(0), uint64(0)
	for _, row := range req.Values {
		// verify values
		var vals []Value
		for _, node := range row {
			ctx := QLEvalContex{}
			qlEval(&ctx, node)
			if ctx.err != nil {
				return added, updated, ctx.err
			}
			vals = append(vals, ctx.out)
		}
		assert(len(vals) == len(req.Names))

		// perform updates
		rec := Record{Cols: req.Names, Vals: vals}
		dbreq := DBSetReq{Record: rec, Mode: req.Mode}
		err := tx.Set(req.Table, &dbreq)
		if err != nil {
			// FIXME: no atomicity for multiple rows
			return added, updated, err
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
	records, err := qlScan(&req.QLScan, &tx.DBReader, nil)
	if err != nil {
		return 0, err
	}

	tdef := getTableDef(&tx.DBReader, req.Table)
	pk := tdef.Cols[:tdef.PKeys]
	for _, rec := range records {
		key := Record{Cols: pk}
		for _, col := range pk {
			key.Vals = append(key.Vals, *rec.Get(col))
		}
		deleted, err := tx.Delete(req.Table, key)
		assert(err == nil && deleted) // deleting an existing row
	}
	return uint64(len(records)), nil
}

// stmt: update
func qlUpdate(req *QLUpdate, tx *DBTX) (uint64, error) {
	// no update to the primary key
	assert(len(req.Names) == len(req.Values))
	tdef := getTableDef(&tx.DBReader, req.Table)
	for _, col := range req.Names {
		ci := colIndex(tdef, col)
		if ci < 0 {
			return 0, fmt.Errorf("unknown column: %s", col)
		}
		if ci < tdef.PKeys {
			return 0, errors.New("cannot update the primary key")
		}
	}

	// old records
	records, err := qlScan(&req.QLScan, &tx.DBReader, nil)
	if err != nil {
		return 0, err
	}

	updated := uint64(0)
	for _, rec := range records {
		// new record
		var newVals []Value
		for _, node := range req.Values {
			ctx := QLEvalContex{env: rec}
			qlEval(&ctx, node)
			if ctx.err != nil {
				return updated, ctx.err
			}
			newVals = append(newVals, ctx.out)
		}
		for i, col := range req.Names {
			rec.Vals[colIndex(tdef, col)] = newVals[i]
		}

		// perform updates
		dbreq := DBSetReq{Record: rec, Mode: MODE_UPDATE_ONLY}
		err := tx.Set(req.Table, &dbreq)
		if err != nil {
			// FIXME: no atomicity for multiple rows
			return updated, err
		}

		// stats
		if dbreq.Updated {
			updated++
		}
	}
	return updated, nil
}

type QLResult struct {
	Records []Record
	Added   uint64
	Updated uint64
	Deleted uint64
}

// execute any statement
func qlExec(w *DBTX, r *DBReader, stmt interface{}) (QLResult, error) {
	result := QLResult{}
	var err error
	switch req := stmt.(type) {
	case *QLSelect:
		result.Records, err = qlSelect(req, r, nil)
		return result, err
	}

	if w == nil {
		return result, errors.New("read-only transaction")
	}

	switch req := stmt.(type) {
	case *QLCreateTable:
		err = qlCreateTable(req, w)
	case *QLInsert:
		result.Added, result.Updated, err = qlInsert(req, w)
	case *QLDelete:
		result.Deleted, err = qlDelete(req, w)
	case *QLUpdate:
		result.Updated, err = qlUpdate(req, w)
	default:
		panic("not implemented")
	}
	return result, err
}
