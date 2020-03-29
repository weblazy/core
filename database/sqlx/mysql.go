package sqlx

import _ "github.com/go-sql-driver/mysql"

const (
	mysqlDriverName = "mysql"

	// because aliyun is using the corba mechanism, so we just use this name to communicate with underlying drivers.
	corbaSql = "corba"
)

func NewMysql(datasource string, opts ...SqlOption) SqlConn {
	return newSqlConn(mysqlDriverName, datasource, opts...)
}

func WithAliyun() SqlOption {
	return func(conn *commonSqlConn) {
		conn.beginTx = beginAliyun
	}
}
