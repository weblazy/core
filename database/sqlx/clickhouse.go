package sqlx

import _ "github.com/kshvakov/clickhouse"

const clickHouseDriverName = "clickhouse"

func NewClickHouse(datasource string, opts ...SqlOption) SqlConn {
	return newSqlConn(clickHouseDriverName, datasource, opts...)
}
