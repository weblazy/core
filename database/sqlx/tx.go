package sqlx

import (
	"context"
	"database/sql"
	"fmt"

	"lazygo/core/errorx"
	"lazygo/core/logx"
)

const (
	disableAutoCommit   = "set autocommit=0"
	enableAutoCommit    = "set autocommit=1"
	registerGlobalTrans = "select last_txc_xid()"
)

type (
	beginnable func(*sql.DB) (trans, error)

	aliyunTx struct {
		txSession
	}

	stdTrans struct {
		txSession
	}

	trans interface {
		Session
		Commit() error
		Rollback() error
	}

	txSession struct {
		tx *sql.Tx
	}
)

func (ct *aliyunTx) Commit() (err error) {
	defer func() {
		if e := ct.tx.Commit(); e != nil {
			if err != nil {
				var be errorx.BatchError
				be = append(be, err, e)
				err = be
			} else {
				err = e
			}
		}
	}()

	return endAliyun(ct.tx, "commit")
}

func (ct *aliyunTx) Rollback() (err error) {
	defer func() {
		if e := ct.tx.Rollback(); e != nil {
			if err != nil {
				var be errorx.BatchError
				be = append(be, err, e)
				err = be
			} else {
				err = e
			}
		}
	}()

	return endAliyun(ct.tx, "rollback")
}

func (t *stdTrans) Commit() error {
	return t.tx.Commit()
}

func (t *stdTrans) Rollback() error {
	return t.tx.Rollback()
}

func (t txSession) Exec(q string, args ...interface{}) (sql.Result, error) {
	return exec(t.tx, q, args...)
}

func (t txSession) Prepare(q string) (StmtSession, error) {
	if stmt, err := t.tx.Prepare(q); err != nil {
		return nil, err
	} else {
		return statement{
			stmt: stmt,
		}, nil
	}
}

func (t txSession) QueryRow(v interface{}, q string, args ...interface{}) error {
	return query(t.tx, func(rows *sql.Rows) error {
		return unmarshalRow(v, rows, true)
	}, q, args...)
}

func (t txSession) QueryRowPartial(v interface{}, q string, args ...interface{}) error {
	return query(t.tx, func(rows *sql.Rows) error {
		return unmarshalRow(v, rows, false)
	}, q, args...)
}

func (t txSession) QueryRows(v interface{}, q string, args ...interface{}) error {
	return query(t.tx, func(rows *sql.Rows) error {
		return unmarshalRows(v, rows, true)
	}, q, args...)
}

func (t txSession) QueryRowsPartial(v interface{}, q string, args ...interface{}) error {
	return query(t.tx, func(rows *sql.Rows) error {
		return unmarshalRows(v, rows, false)
	}, q, args...)
}

func beginAliyun(db *sql.DB) (trans, error) {
	tx, err := db.BeginTx(withCorba(context.Background()), nil)
	if err != nil {
		return nil, err
	}

	logx.Infof("Transaction(%p): %s", tx, disableAutoCommit)
	if _, err := tx.Exec(disableAutoCommit); err != nil {
		return nil, err
	}

	logx.Infof("Transaction(%p): %s", tx, registerGlobalTrans)
	if _, err := tx.Exec(registerGlobalTrans); err != nil {
		return nil, err
	}

	return &aliyunTx{
		txSession: txSession{
			tx: tx,
		},
	}, nil
}

func beginStd(db *sql.DB) (trans, error) {
	if tx, err := db.Begin(); err != nil {
		return nil, err
	} else {
		return &stdTrans{
			txSession: txSession{
				tx: tx,
			},
		}, nil
	}
}

func endAliyun(tx *sql.Tx, cmd string) (err error) {
	defer func() {
		logx.Infof("Transaction(%p): %s", tx, enableAutoCommit)
		if _, e := tx.Exec(enableAutoCommit); e != nil {
			if err != nil {
				var be errorx.BatchError
				be = append(be, err, e)
				err = be
			} else {
				err = e
			}
		}
	}()

	_, err = tx.Exec(cmd)
	return
}

func withCorba(ctx context.Context) context.Context {
	return context.WithValue(ctx, corbaSql, true)
}

func transact(db *commonSqlConn, b beginnable, fn func(Session) error) (err error) {
	conn, err := getSqlConn(db.driverName, db.datasource)
	if err != nil {
		logInstanceError(db.datasource, err)
		return err
	}

	return transactOnConn(conn, b, fn)
}

func transactOnConn(conn *sql.DB, b beginnable, fn func(Session) error) (err error) {
	var tx trans
	tx, err = b(conn)
	if err != nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			err = fmt.Errorf("%#v", p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	return fn(tx)
}
