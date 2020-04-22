package logx

import (
	"encoding/json"
	"fmt"
	"runtime"
)

type (
	Param struct {
		File string      `json:"file"`
		Line int         `json:"line"`
		Data interface{} `json:"data"`
	}
)

func InfoX(args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		param := make(map[string]interface{})
		param["data"] = args
		param["file"] = file
		param["line"] = line

		data, _ := json.Marshal(&Param{
			File: file,
			Line: line,
			Data: args,
		})
		fmt.Printf("%s\n", string(data))
	}
}
