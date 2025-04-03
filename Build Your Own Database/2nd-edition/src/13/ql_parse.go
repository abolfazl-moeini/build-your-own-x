package byodb13

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// syntax tree node types
const (
	QL_UNINIT = 0
	// scalar
	QL_STR = TYPE_BYTES
	QL_I64 = TYPE_INT64
	// binary ops
	QL_CMP_GE = 10 // >=
	QL_CMP_GT = 11 // >
	QL_CMP_LT = 12 // <
	QL_CMP_LE = 13 // <=
	QL_CMP_EQ = 14 // ==
	QL_CMP_NE = 15 // !=
	QL_ADD    = 20
	QL_SUB    = 21
	QL_MUL    = 22
	QL_DIV    = 23
	QL_MOD    = 24
	QL_AND    = 30
	QL_OR     = 31
	// QL_IN     = 32
	// unary ops
	QL_NOT = 50
	QL_NEG = 51
	// others
	QL_SYM  = 100 // column
	QL_TUP  = 101 // tuple
	QL_STAR = 102 // select *
	QL_ERR  = 200 // error; from parsing or evaluation
)

// syntax tree
type QLNode struct {
	Value
	Kids []QLNode
}

// stmt: create table
type QLCreateTable struct {
	Def TableDef
}

// common structure for queries: `INDEX BY`, `FILTER`, `LIMIT`
type QLScan struct {
	Table  string // table name
	Key1   QLNode // index by
	Key2   QLNode
	Filter QLNode // filter
	Offset int64  // limit
	Limit  int64
}

// stmt: select
type QLSelect struct {
	QLScan
	Names  []string // expr AS name
	Output []QLNode
}

// stmt: update
type QLUpdate struct {
	QLScan
	Names  []string
	Values []QLNode
}

// stmt: insert
type QLInsert struct {
	Table  string
	Mode   int // insert | upsert | replace
	Names  []string
	Values [][]QLNode
}

// stmt: delete
type QLDelete struct {
	QLScan
}

type Parser struct {
	input []byte
	idx   int
	err   error
}

func isSpace(ch byte) bool {
	return unicode.IsSpace(rune(ch))
}

func skipSpace(p *Parser) {
	for p.idx < len(p.input) && isSpace(p.input[p.idx]) {
		p.idx++
	}
}

// match multiple keywords sequentially
func pKeyword(p *Parser, kwds ...string) bool {
	save := p.idx
	for _, kw := range kwds {
		skipSpace(p)
		end := p.idx + len(kw)
		if end > len(p.input) {
			p.idx = save
			return false
		}

		// case insensitive matach
		ok := strings.EqualFold(string(p.input[p.idx:end]), kw)
		// token is terminated
		if ok && isSym(kw[len(kw)-1]) && end < len(p.input) {
			ok = !isSym(p.input[end])
		}
		if !ok {
			p.idx = save
			return false
		}

		p.idx += len(kw)
	}
	return true
}

func pErr(p *Parser, format string, args ...interface{}) {
	if p.err == nil {
		p.err = fmt.Errorf(format, args...)
	}
}

func pExpect(p *Parser, tok string, format string, args ...interface{}) {
	if !pKeyword(p, tok) {
		pErr(p, format, args...)
	}
}

func pStmt(p *Parser) (r interface{}) {
	switch {
	case pKeyword(p, "create", "table"):
		r = pCreateTable(p)
	case pKeyword(p, "select"):
		r = pSelect(p)
	case pKeyword(p, "insert", "into"):
		r = pInsert(p, MODE_INSERT_ONLY)
	case pKeyword(p, "replace", "into"):
		r = pInsert(p, MODE_UPDATE_ONLY)
	case pKeyword(p, "upsert", "into"):
		r = pInsert(p, MODE_UPSERT)
	case pKeyword(p, "delete", "from"):
		r = pDelete(p)
	case pKeyword(p, "update"):
		r = pUpdate(p)
	default:
		pErr(p, "unknown stmt")
	}
	if p.err != nil {
		return nil
	}
	return r
}

func pUpdate(p *Parser) *QLUpdate {
	stmt := QLUpdate{}
	stmt.Table = pMustSym(p)

	pExpect(p, "set", "expect `SET`")
	pAssign(p, &stmt)
	for pKeyword(p, ",") {
		pAssign(p, &stmt)
	}

	pScan(p, &stmt.QLScan)
	return &stmt
}

func pAssign(p *Parser, stmt *QLUpdate) {
	stmt.Names = append(stmt.Names, pMustSym(p))
	pExpect(p, "=", "expect `=`")
	stmt.Values = append(stmt.Values, QLNode{})
	pExprOr(p, &stmt.Values[len(stmt.Values)-1])
}

func pDelete(p *Parser) *QLDelete {
	stmt := QLDelete{}
	stmt.Table = pMustSym(p)
	pScan(p, &stmt.QLScan)
	return &stmt
}

func pInsert(p *Parser, mode int) *QLInsert {
	stmt := QLInsert{}
	stmt.Table = pMustSym(p)
	stmt.Mode = mode
	stmt.Names = pNameList(p)

	pExpect(p, "values", "expect `VALUES`")
	stmt.Values = append(stmt.Values, pValueList(p))
	for pKeyword(p, ",") {
		stmt.Values = append(stmt.Values, pValueList(p))
	}
	for _, row := range stmt.Values {
		if len(row) != len(stmt.Names) {
			pErr(p, "values length not match")
		}
	}
	return &stmt
}

func pValueList(p *Parser) []QLNode {
	pExpect(p, "(", "expect value list")

	var vals []QLNode
	comma := true
	for p.err == nil && !pKeyword(p, ")") {
		if !comma {
			pErr(p, "expect comma")
		}
		node := QLNode{}
		pExprOr(p, &node)
		vals = append(vals, node)
		comma = pKeyword(p, ",")
	}
	return vals
}

func pCreateTable(p *Parser) *QLCreateTable {
	stmt := QLCreateTable{}
	stmt.Def.Name = pMustSym(p)
	pExpect(p, "(", "expect parenthesis")

	stmt.Def.Indexes = append(stmt.Def.Indexes, nil) // primary key
	comma := true
	for p.err == nil && !pKeyword(p, ")") {
		if !comma {
			pErr(p, "expect comma")
		}
		switch {
		// index (c1, c2, ...)
		case pKeyword(p, "index"):
			stmt.Def.Indexes = append(stmt.Def.Indexes, pNameList(p))
		// primary key (k1, k2, ...)
		case pKeyword(p, "primary", "key"):
			if stmt.Def.Indexes[0] != nil {
				pErr(p, "duplicate primary key")
			}
			stmt.Def.Indexes[0] = pNameList(p)
		// name type
		default:
			stmt.Def.Cols = append(stmt.Def.Cols, pMustSym(p))
			stmt.Def.Types = append(stmt.Def.Types, pColType(p))
		}
		comma = pKeyword(p, ",") // allowed on the last line
	}
	return &stmt
}

func pColType(p *Parser) uint32 {
	typedef := pMustSym(p)
	switch strings.ToLower(typedef) {
	case "string", "bytes":
		return TYPE_BYTES
	case "int", "int64":
		return TYPE_INT64
	default:
		pErr(p, "bad column type: %s", typedef)
		return 0
	}
}

func pNameList(p *Parser) []string {
	pExpect(p, "(", "expect parenthesis")
	names := []string{pMustSym(p)}
	comma := pKeyword(p, ",")
	for p.err == nil && !pKeyword(p, ")") {
		if !comma {
			pErr(p, "expect comma")
		}
		names = append(names, pMustSym(p))
		comma = pKeyword(p, ",")
	}
	return names
}

func pMustSym(p *Parser) string {
	name := QLNode{}
	if !pSym(p, &name) {
		pErr(p, "expect name")
	}
	return string(name.Str)
}

func pSelect(p *Parser) *QLSelect {
	// SELECT xxx
	stmt := QLSelect{}
	pSelectExprList(p, &stmt)
	// FROM table
	pExpect(p, "from", "expect `FROM` table")
	stmt.Table = pMustSym(p)
	// INDEX BY xxx FILTER yyy LIMIT zzz
	pScan(p, &stmt.QLScan)
	return &stmt
}

func pSelectExprList(p *Parser, node *QLSelect) {
	pSelectExpr(p, node)
	for pKeyword(p, ",") {
		pSelectExpr(p, node)
	}
}

func pSelectExpr(p *Parser, node *QLSelect) {
	if pKeyword(p, "*") {
		node.Names = append(node.Names, "*")
		node.Output = append(node.Output, QLNode{Value: Value{Type: QL_STAR}})
		return
	}

	node.Output = append(node.Output, QLNode{})
	pExprOr(p, &node.Output[len(node.Output)-1])
	node.Names = append(node.Names, "")
	if pKeyword(p, "as") {
		node.Names[len(node.Names)-1] = pMustSym(p)
	}
}

func pScan(p *Parser, node *QLScan) {
	// INDEX BY ...
	if pKeyword(p, "index", "by") {
		pIndexBy(p, node)
	}
	// FILTER condition
	if pKeyword(p, "filter") {
		pExprOr(p, &node.Filter)
	}
	// LIMIT x, y
	node.Offset, node.Limit = 0, math.MaxInt64
	if pKeyword(p, "limit") {
		pLimit(p, node)
	}
}

func pIndexBy(p *Parser, node *QLScan) {
	// INDEX BY cols < vals
	// INDEX BY cols < vals AND cols > vals
	index := QLNode{}
	pExprAnd(p, &index)
	if index.Type == QL_AND {
		node.Key1, node.Key2 = index.Kids[0], index.Kids[1]
	} else {
		node.Key1 = index
	}
	pVerifyScanKey(p, &node.Key1)
	if node.Key2.Type != 0 {
		pVerifyScanKey(p, &node.Key2)
	}
	if node.Key1.Type == QL_CMP_EQ && node.Key2.Type != 0 {
		pErr(p, "bad `INDEX BY`: expect only a single `=`")
	}
}

func pVerifyScanKey(p *Parser, node *QLNode) {
	switch node.Type {
	case QL_CMP_EQ, QL_CMP_GE, QL_CMP_GT, QL_CMP_LT, QL_CMP_LE:
	default:
		pErr(p, "bad `INDEX BY`: not a comparison")
		return
	}

	left, right := node.Kids[0], node.Kids[1]
	if left.Type != QL_TUP && right.Type != QL_TUP {
		left = QLNode{Value: Value{Type: QL_TUP}, Kids: []QLNode{left}}
		right = QLNode{Value: Value{Type: QL_TUP}, Kids: []QLNode{right}}
	}
	if left.Type != QL_TUP || right.Type != QL_TUP {
		pErr(p, "bad `INDEX BY`: bad comparison")
	}
	if len(left.Kids) != len(right.Kids) {
		pErr(p, "bad `INDEX BY`: bad comparison")
	}

	for _, name := range left.Kids {
		if name.Type != QL_SYM {
			pErr(p, "bad `INDEX BY`: expect column name")
		}
	}

	node.Kids[0], node.Kids[1] = left, right
}

func pLimit(p *Parser, node *QLScan) {
	// LIMIT count
	// LIMIT offset, count
	offset, count := QLNode{}, QLNode{}
	ok := pNum(p, &count)
	if pKeyword(p, ",") {
		offset = count
		ok = ok && pNum(p, &count)
	}
	if !ok || offset.I64 < 0 || count.I64 < 0 || offset.I64+count.I64 < 0 {
		pErr(p, "bad `LIMIT`")
	}
	// output
	node.Offset = offset.I64
	if count.Type != 0 {
		node.Limit = node.Offset + count.I64
	}
}

func pExprTuple(p *Parser, node *QLNode) {
	kids := []QLNode{}
	comma := true
	for p.err == nil && !pKeyword(p, ")") {
		if !comma {
			pErr(p, "expect comma")
		}
		kids = append(kids, QLNode{})
		pExprOr(p, &kids[len(kids)-1])
		comma = pKeyword(p, ",")
	}

	if len(kids) == 1 && !comma {
		*node = kids[0] // just a pair of parentheses
	} else {
		node.Type = QL_TUP
		node.Kids = kids
	}
}

func pExprOr(p *Parser, node *QLNode) {
	pExprBinop(p, node, []string{"or"}, []uint32{QL_OR}, pExprAnd)
}

func pExprAnd(p *Parser, node *QLNode) {
	pExprBinop(p, node, []string{"and"}, []uint32{QL_AND}, pExprNot)
}

func pExprNot(p *Parser, node *QLNode) {
	switch {
	case pKeyword(p, "not"):
		node.Type = QL_NOT
		node.Kids = []QLNode{{}}
		pExprCmp(p, &node.Kids[0])
	default:
		pExprCmp(p, node)
	}
}

func pExprCmp(p *Parser, node *QLNode) {
	pExprBinop(p, node,
		[]string{
			"=", "!=",
			">=", "<=", ">", "<",
		},
		[]uint32{
			QL_CMP_EQ, QL_CMP_NE,
			QL_CMP_GE, QL_CMP_LE, QL_CMP_GT, QL_CMP_LT,
		},
		pExprAdd)
}

func pExprAdd(p *Parser, node *QLNode) {
	pExprBinop(p, node,
		[]string{"+", "-"}, []uint32{QL_ADD, QL_SUB}, pExprMul)
}

func pExprMul(p *Parser, node *QLNode) {
	pExprBinop(p, node,
		[]string{"*", "/", "%"}, []uint32{QL_MUL, QL_DIV, QL_MOD}, pExprUnop)
}

func pExprBinop(
	p *Parser, node *QLNode,
	ops []string, types []uint32, next func(*Parser, *QLNode),
) {
	assert(len(ops) == len(types))
	left := QLNode{}
	next(p, &left)

	for {
		i := 0
		for i < len(ops) && !pKeyword(p, ops[i]) {
			i++
		}
		if i >= len(ops) {
			*node = left
			return
		}
		new := QLNode{Value: Value{Type: types[i]}}
		new.Kids = []QLNode{left, {}}
		next(p, &new.Kids[1])
		left = new
	}
}

func pExprUnop(p *Parser, node *QLNode) {
	switch {
	case pKeyword(p, "-"):
		node.Type = QL_NEG
		node.Kids = []QLNode{{}}
		pExprAtom(p, &node.Kids[0])
	default:
		pExprAtom(p, node)
	}
}

func pExprAtom(p *Parser, node *QLNode) {
	switch {
	case pKeyword(p, "("):
		pExprTuple(p, node)
	case pSym(p, node):
	case pNum(p, node):
	case pStr(p, node):
	default:
		pErr(p, "expect symbol, number or string")
	}
}

var pKeywordSet = map[string]bool{
	"from":   true,
	"index":  true,
	"filter": true,
	"limit":  true,
}

func isSym(ch byte) bool {
	r := rune(ch)
	return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_'
}

func isSymStart(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_' || ch == '@'
}

// TODO: quoted symbol
func pSym(p *Parser, node *QLNode) bool {
	skipSpace(p)

	end := p.idx
	if !(end < len(p.input) && isSymStart(p.input[end])) {
		return false
	}
	end++
	for end < len(p.input) && isSym(p.input[end]) {
		end++
	}
	if pKeywordSet[strings.ToLower(string(p.input[p.idx:end]))] {
		return false // not allowed
	}

	node.Type = QL_SYM
	node.Str = p.input[p.idx:end]
	p.idx = end
	return true
}

func pNum(p *Parser, node *QLNode) bool {
	skipSpace(p)

	end := p.idx
	for end < len(p.input) && unicode.IsNumber(rune(p.input[end])) {
		end++
	}
	if end == p.idx {
		return false
	}
	if end < len(p.input) && isSym(p.input[end]) {
		return false
	}

	i64, err := strconv.ParseUint(string(p.input[p.idx:end]), 10, 64)
	if err != nil || i64 > math.MaxInt64 {
		return false
	}

	node.Type = QL_I64
	node.I64 = int64(i64)
	p.idx = end
	return true
}

func pStr(p *Parser, node *QLNode) bool {
	skipSpace(p)

	cur := p.idx
	quote := byte(0)
	if cur < len(p.input) {
		quote = p.input[cur]
		cur++
	}
	if !(quote == '"' || quote == '\'') {
		return false
	}

	var s []byte
	for cur < len(p.input) && p.input[cur] != quote {
		switch p.input[cur] {
		case '\\':
			cur++
			if cur >= len(p.input) {
				pErr(p, "string not terminated")
				return false
			}
			switch p.input[cur] {
			case '"', '\'', '\\':
				s = append(s, p.input[cur])
				cur++
			// TODO: more
			default:
				pErr(p, "unknown escape")
				return false
			}
		default:
			s = append(s, p.input[cur])
			cur++
		}
	}

	if !(cur < len(p.input) && p.input[cur] == quote) {
		pErr(p, "string not terminated")
		return false
	}
	cur++

	node.Type = QL_STR
	node.Str = s
	p.idx = cur
	return true
}
