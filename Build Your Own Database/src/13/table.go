package byodb13

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

type DB struct {
	Path string
	// internals
	kv     KV
	mu     sync.Mutex
	tables map[string]*TableDef // cached table definition
}

// read-only DB transactions
type DBReader struct {
	kv KVTX // contains the KVReader
	db *DB
}

func (db *DB) BeginRead(tx *DBReader) {
	tx.db = db
	db.kv.BeginRead(&tx.kv.KVReader)
}
func (db *DB) EndRead(tx *DBReader) {
	db.kv.EndRead(&tx.kv.KVReader)
}

// DB transactions
type DBTX struct {
	DBReader
}

func (db *DB) Begin(tx *DBTX) {
	tx.db = db
	db.kv.Begin(&tx.kv)
}
func (db *DB) Commit(tx *DBTX) error {
	return db.kv.Commit(&tx.kv)
}
func (db *DB) Abort(tx *DBTX) {
	db.kv.Abort(&tx.kv)
}

// table definition
type TableDef struct {
	// user defined
	Name    string
	Types   []uint32 // column types
	Cols    []string // column names
	PKeys   int      // the first `PKeys` columns are the primary key
	Indexes [][]string
	// auto-assigned B-tree key prefixes for different tables/indexes
	Prefix        uint32
	IndexPrefixes []uint32
}

const (
	TYPE_ERROR = 0
	TYPE_BYTES = 1
	TYPE_INT64 = 2
)

// table cell
type Value struct {
	Type uint32
	I64  int64
	Str  []byte
}

// table row
type Record struct {
	Cols []string
	Vals []Value
}

func (rec *Record) AddStr(key string, val []byte) *Record {
	rec.Cols = append(rec.Cols, key)
	rec.Vals = append(rec.Vals, Value{Type: TYPE_BYTES, Str: val})
	return rec
}
func (rec *Record) AddInt64(key string, val int64) *Record {
	rec.Cols = append(rec.Cols, key)
	rec.Vals = append(rec.Vals, Value{Type: TYPE_INT64, I64: val})
	return rec
}

func (rec *Record) Get(key string) *Value {
	for i, c := range rec.Cols {
		if c == key {
			return &rec.Vals[i]
		}
	}
	return nil
}

// rearrange a record to the defined column order
func reorderRecord(tdef *TableDef, rec Record) ([]Value, error) {
	assert(len(rec.Cols) == len(rec.Vals))
	out := make([]Value, len(tdef.Cols))
	for i, c := range tdef.Cols {
		v := rec.Get(c)
		if v == nil {
			continue // leave this column uninitialized
		}
		if v.Type != tdef.Types[i] {
			return nil, fmt.Errorf("bad column type: %s", c)
		}
		out[i] = *v
	}
	return out, nil
}

func valuesComplete(tdef *TableDef, vals []Value, n int) error {
	for i, v := range vals {
		if i < n && v.Type == 0 {
			return fmt.Errorf("missing column: %s", tdef.Cols[i])
		} else if i >= n && v.Type != 0 {
			return fmt.Errorf("extra column: %s", tdef.Cols[i])
		}
	}
	return nil
}

// reorder a record and check for missing columns.
// n == tdef.PKeys: record is exactly a primary key
// n == len(tdef.Cols): record contains all columns
func checkRecord(tdef *TableDef, rec Record, n int) ([]Value, error) {
	vals, err := reorderRecord(tdef, rec)
	if err != nil {
		return nil, err
	}
	err = valuesComplete(tdef, vals, n)
	if err != nil {
		return nil, err
	}
	return vals, nil
}

// check column types
func checkRecordTypes(tdef *TableDef, rec Record) error {
	for i, c := range rec.Cols {
		j := colIndex(tdef, c)
		if j < 0 || tdef.Types[j] != rec.Vals[i].Type {
			return fmt.Errorf("bad column: %s", c)
		}
	}
	return nil
}

// 1. strings are encoded as null-terminated strings,
//    escape the null byte so that strings contain no null byte.
// 2. "\xff" represents the highest order in key comparisons,
//    also escape the first byte if it's 0xff.
func escapeString(in []byte) []byte {
	first := len(in) > 0 && in[0] >= 0xfe
	zeros := bytes.Count(in, []byte{0})
	ones := bytes.Count(in, []byte{1})
	if !first && zeros+ones == 0 {
		return in
	}

	nescape := zeros + ones
	if first {
		nescape++
	}
	out := make([]byte, len(in)+nescape)

	pos := 0
	if first {
		out[0] = 0xfe
		out[1] = in[0]
		pos += 2
		in = in[1:]
	}

	for _, ch := range in {
		if ch <= 1 {
			out[pos+0] = 0x01
			out[pos+1] = ch + 1
			pos += 2
		} else {
			out[pos] = ch
			pos += 1
		}
	}
	return out
}

func unescapeString(in []byte) []byte {
	assert(len(in) == 0 || in[0] != 0xff)
	first := len(in) > 0 && in[0] == 0xfe
	if !first && bytes.Count(in, []byte{1}) == 0 {
		return in
	}

	out := make([]byte, len(in))
	pos := 0
	if first {
		out[0] = in[1]
		pos++
		in = in[2:]
	}
	for i := 0; i < len(in); i++ {
		if in[i] == 0x01 {
			i++
			assert(in[i] >= 1)
			out[pos] = in[i] - 1
		} else {
			out[pos] = in[i]
		}
		pos++
	}
	return out[:pos]
}

// order-preserving encoding
func encodeValues(out []byte, vals []Value) []byte {
	for _, v := range vals {
		switch v.Type {
		case TYPE_INT64:
			var buf [8]byte
			u := uint64(v.I64) + (1 << 63)
			binary.BigEndian.PutUint64(buf[:], u)
			out = append(out, buf[:]...)
		case TYPE_BYTES:
			out = append(out, escapeString(v.Str)...)
			out = append(out, 0) // null-terminated
		default:
			panic("what?")
		}
	}
	return out
}

// for primary keys or index keys
func encodeKey(out []byte, prefix uint32, vals []Value) []byte {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], prefix)
	out = append(out, buf[:]...)
	out = encodeValues(out, vals)
	return out
}

func decodeValues(in []byte, out []Value) {
	for i := range out {
		switch out[i].Type {
		case TYPE_INT64:
			u := binary.BigEndian.Uint64(in[:8])
			out[i].I64 = int64(u - (1 << 63))
			in = in[8:]
		case TYPE_BYTES:
			idx := bytes.IndexByte(in, 0)
			assert(idx >= 0)
			out[i].Str = unescapeString(in[:idx:idx])
			in = in[idx+1:]
		default:
			panic("what?")
		}
	}
	assert(len(in) == 0)
}

// get a single row by the primary key
func dbGet(tx *DBReader, tdef *TableDef, rec *Record) (bool, error) {
	// just a shortcut for the scan operation
	sc := Scanner{
		Cmp1: CMP_GE,
		Cmp2: CMP_LE,
		Key1: *rec,
		Key2: *rec,
	}
	if err := dbScan(tx, tdef, &sc); err != nil {
		return false, err
	}
	if sc.Valid() {
		sc.Deref(rec)
		sc.Next()
		assert(!sc.Valid()) // incomplete key
		return true, nil
	} else {
		return false, nil
	}
}

// internal table: metadata
var TDEF_META = &TableDef{
	Prefix: 1,
	Name:   "@meta",
	Types:  []uint32{TYPE_BYTES, TYPE_BYTES},
	Cols:   []string{"key", "val"},
	PKeys:  1,
}

// internal table: table schemas
var TDEF_TABLE = &TableDef{
	Prefix: 2,
	Name:   "@table",
	Types:  []uint32{TYPE_BYTES, TYPE_BYTES},
	Cols:   []string{"name", "def"},
	PKeys:  1,
}

var INTERNAL_TABLES map[string]*TableDef = map[string]*TableDef{
	"@meta":  TDEF_META,
	"@table": TDEF_TABLE,
}

// get the table definition by name
func getTableDef(tx *DBReader, name string) *TableDef {
	if tdef, ok := INTERNAL_TABLES[name]; ok {
		return tdef // expose internal tables
	}

	db := tx.db
	db.mu.Lock()
	tdef, ok := db.tables[name]
	db.mu.Unlock()

	if !ok {
		tdef = getTableDefDB(tx, name)
		db.mu.Lock()
		if db.tables == nil {
			db.tables = map[string]*TableDef{}
		}
		if tdef != nil {
			db.tables[name] = tdef
		}
		db.mu.Unlock()
	}
	return tdef
}

func getTableDefDB(tx *DBReader, name string) *TableDef {
	rec := (&Record{}).AddStr("name", []byte(name))
	ok, err := dbGet(tx, TDEF_TABLE, rec)
	assert(err == nil)
	if !ok {
		return nil
	}

	tdef := &TableDef{}
	err = json.Unmarshal(rec.Get("def").Str, tdef)
	assert(err == nil)
	return tdef
}

// get a single row by the primary key
func (tx *DBReader) Get(table string, rec *Record) (bool, error) {
	tdef := getTableDef(tx, table)
	if tdef == nil {
		return false, fmt.Errorf("table not found: %s", table)
	}

	// check and reorder the primary key
	values, err := checkRecord(tdef, *rec, tdef.PKeys)
	if err != nil {
		return false, err
	}
	rec.Cols = tdef.Cols[:tdef.PKeys]
	rec.Vals = values[:tdef.PKeys]

	return dbGet(tx, tdef, rec)
}

const TABLE_PREFIX_MIN = 100

func tableDefCheck(tdef *TableDef) error {
	// verify the table definition
	bad := tdef.Name == "" || len(tdef.Cols) == 0
	bad = bad || len(tdef.Cols) != len(tdef.Types)
	bad = bad || !(1 <= tdef.PKeys && int(tdef.PKeys) <= len(tdef.Cols))
	if bad {
		return fmt.Errorf("bad table definition: %s", tdef.Name)
	}
	// verify the indexes
	for i, index := range tdef.Indexes {
		index, err := checkIndexKeys(tdef, index)
		if err != nil {
			return err
		}
		tdef.Indexes[i] = index
	}
	return nil
}

func checkIndexKeys(tdef *TableDef, index []string) ([]string, error) {
	icols := map[string]bool{}
	for _, c := range index {
		// check the index columns
		if colIndex(tdef, c) < 0 {
			return nil, fmt.Errorf("unknown index column: %s", c)
		}
		if icols[c] {
			return nil, fmt.Errorf("duplicated column in index: %s", c)
		}
		icols[c] = true
	}
	// add the primary key to the index
	for _, c := range tdef.Cols[:tdef.PKeys] {
		if !icols[c] {
			index = append(index, c)
		}
	}
	assert(len(index) < len(tdef.Cols))
	return index, nil
}

// create a new table
func (tx *DBTX) TableNew(tdef *TableDef) error {
	if err := tableDefCheck(tdef); err != nil {
		return err
	}

	// check the existing table
	table := (&Record{}).AddStr("name", []byte(tdef.Name))
	ok, err := dbGet(&tx.DBReader, TDEF_TABLE, table)
	assert(err == nil)
	if ok {
		return fmt.Errorf("table exists: %s", tdef.Name)
	}

	// allocate new prefixes
	assert(tdef.Prefix == 0)
	tdef.Prefix = TABLE_PREFIX_MIN
	meta := (&Record{}).AddStr("key", []byte("next_prefix"))
	ok, err = dbGet(&tx.DBReader, TDEF_META, meta)
	assert(err == nil)
	if ok {
		tdef.Prefix = binary.LittleEndian.Uint32(meta.Get("val").Str)
		assert(tdef.Prefix > TABLE_PREFIX_MIN)
	} else {
		meta.AddStr("val", make([]byte, 4))
	}
	for i := range tdef.Indexes {
		prefix := tdef.Prefix + 1 + uint32(i)
		tdef.IndexPrefixes = append(tdef.IndexPrefixes, prefix)
	}

	// update the next prefix
	ntree := 1 + uint32(len(tdef.Indexes))
	binary.LittleEndian.PutUint32(meta.Get("val").Str, tdef.Prefix+ntree)
	err = dbUpdate(tx, TDEF_META, &DBSetReq{Record: *meta})
	if err != nil {
		return err
	}

	// store the definition
	if tdef.Indexes == nil {
		tdef.Indexes = [][]string{}
	}
	if tdef.IndexPrefixes == nil {
		tdef.IndexPrefixes = []uint32{}
	}
	val, err := json.Marshal(tdef)
	assert(err == nil)
	table.AddStr("def", val)
	err = dbUpdate(tx, TDEF_TABLE, &DBSetReq{Record: *table})
	return err
}

type DBSetReq struct {
	// in
	Record Record
	Mode   int
	// out
	Updated bool
	Added   bool
}

// add a row to the table
// FIXME: check key length
func dbUpdate(tx *DBTX, tdef *TableDef, dbreq *DBSetReq) error {
	values, err := checkRecord(tdef, dbreq.Record, len(tdef.Cols))
	if err != nil {
		return err
	}

	// by the primary key
	key := encodeKey(nil, tdef.Prefix, values[:tdef.PKeys])
	val := encodeValues(nil, values[tdef.PKeys:])
	req := InsertReq{Key: key, Val: val, Mode: dbreq.Mode}
	_ = tx.kv.Update(&req)

	// stats
	dbreq.Added, dbreq.Updated = req.Added, req.Updated
	if !req.Updated || len(tdef.Indexes) == 0 {
		return nil
	}

	// maintain indexes
	if req.Updated && !req.Added {
		decodeValues(req.Old, values[tdef.PKeys:]) // get the old row
		indexOp(tx, tdef, Record{tdef.Cols, values}, INDEX_DEL)
	}
	if req.Updated {
		indexOp(tx, tdef, dbreq.Record, INDEX_ADD)
	}
	return nil
}

const (
	INDEX_ADD = 1
	INDEX_DEL = 2
)

// maintain indexes after a record is added or removed
func indexOp(tx *DBTX, tdef *TableDef, rec Record, op int) {
	key := make([]byte, 0, 256)
	irec := make([]Value, len(tdef.Cols))
	for i, index := range tdef.Indexes {
		// the indexed key
		for j, c := range index {
			irec[j] = *rec.Get(c)
		}
		// update the KV store
		key = encodeKey(key[:0], tdef.IndexPrefixes[i], irec[:len(index)])
		done := false
		switch op {
		case INDEX_ADD:
			done = tx.kv.Update(&InsertReq{Key: key})
		case INDEX_DEL:
			done = tx.kv.Del(&DeleteReq{Key: key})
		default:
			panic("what?")
		}
		assert(done)
	}
}

// add a record
func (tx *DBTX) Set(table string, req *DBSetReq) error {
	tdef := getTableDef(&tx.DBReader, table)
	if tdef == nil {
		return fmt.Errorf("table not found: %s", table)
	}
	return dbUpdate(tx, tdef, req)
}
func (tx *DBTX) Insert(table string, rec Record) (bool, error) {
	req := DBSetReq{Record: rec, Mode: MODE_INSERT_ONLY}
	err := tx.Set(table, &req)
	return req.Added, err
}
func (tx *DBTX) Update(table string, rec Record) (bool, error) {
	req := DBSetReq{Record: rec, Mode: MODE_UPDATE_ONLY}
	err := tx.Set(table, &req)
	return req.Added, err
}
func (tx *DBTX) Upsert(table string, rec Record) (bool, error) {
	req := DBSetReq{Record: rec, Mode: MODE_UPSERT}
	err := tx.Set(table, &req)
	return req.Added, err
}

// delete a record by its primary key
// FIXME: check key length
func dbDelete(tx *DBTX, tdef *TableDef, rec Record) (bool, error) {
	values, err := checkRecord(tdef, rec, tdef.PKeys)
	if err != nil {
		return false, err
	}

	// delete the record
	key := encodeKey(nil, tdef.Prefix, values[:tdef.PKeys])
	req := DeleteReq{Key: key}
	deleted := tx.kv.Del(&req)
	if !deleted || len(tdef.Indexes) == 0 {
		return deleted, nil
	}

	// maintain indexes
	if deleted {
		for i := tdef.PKeys; i < len(tdef.Types); i++ {
			values[i].Type = tdef.Types[i]
		}
		decodeValues(req.Old, values[tdef.PKeys:]) // get the old row
		indexOp(tx, tdef, Record{tdef.Cols, values}, INDEX_DEL)
	}
	return true, nil
}

// remove a record
func (tx *DBTX) Delete(table string, rec Record) (bool, error) {
	tdef := getTableDef(&tx.DBReader, table)
	if tdef == nil {
		return false, fmt.Errorf("table not found: %s", table)
	}
	return dbDelete(tx, tdef, rec)
}

func (db *DB) Open() error {
	db.kv.Path = db.Path
	return db.kv.Open()
}

func (db *DB) Close() {
	db.kv.Close()
}

// the iterator for range queries
type Scanner struct {
	// the range, from Key1 to Key2
	Cmp1 int // CMP_??
	Cmp2 int // optional
	Key1 Record
	Key2 Record // optional
	// internal
	tx      *DBReader
	tdef    *TableDef
	indexNo int    // -1: use the primary key; >= 0: use an index
	iter    *BIter // the underlying B-tree iterator
	keyEnd  []byte // the encoded Key2
}

// within the range or not?
func (sc *Scanner) Valid() bool {
	if !sc.iter.Valid() {
		return false
	}
	key, _ := sc.iter.Deref()
	return cmpOK(key, sc.Cmp2, sc.keyEnd)
}

// move the underlying B-tree iterator
func (sc *Scanner) Next() {
	assert(sc.Valid())
	if sc.Cmp1 > 0 {
		sc.iter.Next()
	} else {
		sc.iter.Prev()
	}
}

// fetch the current row
func (sc *Scanner) Deref(rec *Record) {
	assert(sc.Valid())

	tdef := sc.tdef
	rec.Cols = tdef.Cols
	rec.Vals = rec.Vals[:0]
	key, val := sc.iter.Deref()

	if sc.indexNo < 0 {
		// primary key, decode the KV pair
		for _, type_ := range tdef.Types {
			rec.Vals = append(rec.Vals, Value{Type: type_})
		}
		decodeValues(key[4:], rec.Vals[:tdef.PKeys])
		decodeValues(val, rec.Vals[tdef.PKeys:])
	} else {
		// secondary index
		assert(len(val) == 0)

		// decode the primary key first
		index := tdef.Indexes[sc.indexNo]
		ival := make([]Value, len(index))
		for i, c := range index {
			ival[i].Type = tdef.Types[colIndex(tdef, c)]
		}
		decodeValues(key[4:], ival)
		icol := Record{index, ival}

		// fetch the row by the primary key
		rec.Cols = tdef.Cols[:tdef.PKeys]
		for _, c := range rec.Cols {
			rec.Vals = append(rec.Vals, *icol.Get(c))
		}
		// TODO: skip this if the index contains all the columns
		ok, err := dbGet(sc.tx, tdef, rec)
		assert(ok && err == nil)
	}
}

func isPrefix(long []string, short []string) bool {
	if len(long) < len(short) {
		return false
	}
	for i, c := range short {
		if long[i] != c {
			return false
		}
	}
	return true
}

func findIndex(tdef *TableDef, keys []string) (int, error) {
	pk := tdef.Cols[:tdef.PKeys]
	if isPrefix(pk, keys) {
		// use the primary key.
		// also works for full table scans without a key.
		return -1, nil
	}

	// find a suitable index
	winner := -2
	for i, index := range tdef.Indexes {
		if !isPrefix(index, keys) {
			continue
		}
		if winner == -2 || len(index) < len(tdef.Indexes[winner]) {
			winner = i
		}
	}
	if winner == -2 {
		return -2, fmt.Errorf("no index found")
	}
	return winner, nil
}

func colIndex(tdef *TableDef, col string) int {
	for i, c := range tdef.Cols {
		if c == col {
			return i
		}
	}
	return -1
}

// The range key can be a prefix of the index key,
// we may have to encode missing columns to make the comparison work.
func encodeKeyPartial(
	out []byte, prefix uint32, values []Value,
	tdef *TableDef, keys []string, cmp int,
) []byte {
	out = encodeKey(out, prefix, values)

	// Encode the missing columns as either minimum or maximum values,
	// depending on the comparison operator.
	// 1. The empty string is lower than all possible value encodings,
	//    thus we don't need to add anything for CMP_LT and CMP_GE.
	// 2. The maximum encodings are all 0xff bytes.
	max := cmp == CMP_GT || cmp == CMP_LE
loop:
	for i := len(values); max && i < len(keys); i++ {
		switch tdef.Types[colIndex(tdef, keys[i])] {
		case TYPE_BYTES:
			out = append(out, 0xff)
			break loop // stops here since no string encoding starts with 0xff
		case TYPE_INT64:
			out = append(out, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff)
		default:
			panic("what?")
		}
	}
	return out
}

func dbScan(tx *DBReader, tdef *TableDef, req *Scanner) error {
	// sanity checks
	switch {
	case req.Cmp1 > 0 && req.Cmp2 < 0:
	case req.Cmp2 > 0 && req.Cmp1 < 0:
	case req.Cmp1 != 0 && req.Cmp2 == 0 && len(req.Key2.Cols) == 0:
	default:
		return fmt.Errorf("bad range")
	}
	if req.Cmp2 != 0 && !reflect.DeepEqual(req.Key1.Cols, req.Key2.Cols) {
		return fmt.Errorf("bad range key")
	}
	if err := checkRecordTypes(tdef, req.Key1); err != nil {
		return err
	}
	if req.Cmp2 != 0 {
		if err := checkRecordTypes(tdef, req.Key2); err != nil {
			return err
		}
	}

	// select an index
	indexNo, err := findIndex(tdef, req.Key1.Cols)
	if err != nil {
		return err
	}
	index, prefix := tdef.Cols[:tdef.PKeys], tdef.Prefix
	if indexNo >= 0 {
		index, prefix = tdef.Indexes[indexNo], tdef.IndexPrefixes[indexNo]
	}

	req.tx = tx
	req.tdef = tdef
	req.indexNo = indexNo

	// seek to the start key
	keyStart := encodeKeyPartial(
		nil, prefix, req.Key1.Vals, tdef, index, req.Cmp1)
	req.iter = tx.kv.Seek(keyStart, req.Cmp1)

	// the end key
	if req.Cmp2 == 0 {
		// no end; the range is bounded by the prefix
		switch req.Cmp1 {
		case CMP_GE, CMP_GT:
			req.Cmp2 = CMP_LT
			req.keyEnd = encodeKey(nil, prefix+1, nil)
		case CMP_LE, CMP_LT:
			req.Cmp2 = CMP_GT
			req.keyEnd = encodeKey(nil, prefix, nil)
		default:
			panic("unreachable")
		}
	} else {
		req.keyEnd = encodeKeyPartial(
			nil, prefix, req.Key2.Vals, tdef, index, req.Cmp2)
	}
	return nil
}

// range query
func (tx *DBReader) Scan(table string, req *Scanner) error {
	tdef := getTableDef(tx, table)
	if tdef == nil {
		return fmt.Errorf("table not found: %s", table)
	}
	return dbScan(tx, tdef, req)
}
