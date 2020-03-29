package tpcluster

import (
	tp "github.com/weblazy/teleport"
)

var (
	StatusUnauthorized = tp.NewStatus(tp.CodeUnauthorized, "auth faild", nil)
)
