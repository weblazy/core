package sqlx

import _ "github.com/lib/pq"

const postgreDriverName = "postgres"

func NewPostgre(datasource string, opts ...SqlOption) SqlConn {
	return newSqlConn(postgreDriverName, datasource, opts...)
}
