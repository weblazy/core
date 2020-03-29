package sqlx

import (
	"database/sql"
	"strings"
	"time"

	"lazygo/core/executors"
	"lazygo/core/logx"
)

const (
	flushInterval = time.Second
	maxBulkRows   = 1000
)

type (
	ResultHandler func(sql.Result, error)

	BulkInserter struct {
		executor *executors.PeriodicalExecutor
		inserter *dbInserter
	}
)

func NewBulkInserter(sqlConn SqlConn, stmt string) *BulkInserter {
	inserter := &dbInserter{
		sqlConn:           sqlConn,
		stmtWithoutValues: stmt,
	}

	return &BulkInserter{
		executor: executors.NewPeriodicalExecutor(flushInterval, inserter),
		inserter: inserter,
	}
}

func (bi *BulkInserter) Flush() {
	bi.executor.ForceFlush()
}

func (bi *BulkInserter) Insert(valueFormat string, args ...interface{}) error {
	value, err := format(valueFormat, args...)
	if err != nil {
		return err
	}

	bi.executor.Add(value)

	return nil
}

func (bi *BulkInserter) SetResultHandler(handler ResultHandler) {
	bi.executor.Sync(func() {
		bi.inserter.resultHandler = handler
	})
}

func (bi *BulkInserter) UpdateOrDelete(fn func()) {
	bi.executor.ForceFlush()
	fn()
}

func (bi *BulkInserter) UpdateStmt(stmt string) {
	bi.executor.ForceFlush()
	bi.executor.Sync(func() {
		bi.inserter.stmtWithoutValues = stmt
	})
}

type (
	stmtValuesPair struct {
		stmt   string
		values []string
	}

	dbInserter struct {
		sqlConn           SqlConn
		stmtWithoutValues string
		values            []string
		resultHandler     ResultHandler
	}
)

func (in *dbInserter) AddTask(task interface{}) bool {
	in.values = append(in.values, task.(string))
	return len(in.values) >= maxBulkRows
}

func (in *dbInserter) Execute(bulk interface{}) {
	pair := bulk.(stmtValuesPair)
	values := pair.values
	if len(values) == 0 {
		return
	}

	stmtWithoutValues := pair.stmt
	valuesStr := strings.Join(values, ", ")
	stmt := strings.Join([]string{stmtWithoutValues, valuesStr}, " ")
	result, err := in.sqlConn.Exec(stmt)
	if in.resultHandler != nil {
		in.resultHandler(result, err)
	} else if err != nil {
		logx.Error(err)
	}
}

func (in *dbInserter) RemoveAll() interface{} {
	values := in.values
	in.values = nil
	return stmtValuesPair{
		stmt:   in.stmtWithoutValues,
		values: values,
	}
}
