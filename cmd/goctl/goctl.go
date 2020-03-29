package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

var (
	service   = ""
	pack      = ""
	funcArr   = make([][]string, 0)
	protoFile = flag.String("p", "", "The protoFile")
)

func main() {
	flag.Parse()
	fmt.Println(*protoFile)
	f, err := os.Open(*protoFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	rd := bufio.NewReader(f)
	for {
		line, err := rd.ReadString('\n') //以'\n'为结束符读入一行

		if err != nil || io.EOF == err {
			break
		}
		dealPackage(line)
		dealService(line)
		dealFunc(line)
	}
	content := combine()
	WriteWithIoutil(strings.ToLower(service)+".go", content)
}

func dealPackage(line string) {
	r := regexp.MustCompile("\\s*package\\s+(\\w+)")
	result := r.FindAllStringSubmatch(line, -1)
	if len(result) > 0 {
		pack = result[0][1]
	}
}

func dealService(line string) {
	r := regexp.MustCompile("\\s*service\\s+(\\w+)\\s")
	result := r.FindAllStringSubmatch(line, -1)
	if len(result) > 0 {
		service = result[0][1]
	}
}

func dealFunc(line string) {
	r := regexp.MustCompile("\\s*rpc\\s+(\\w+)\\(\\s*(\\w+)\\s*\\)\\s*returns\\s*\\(\\s*(\\w+)\\s*\\).*")
	result := r.FindAllStringSubmatch(line, -1)
	if len(result) > 0 {
		funcArr = append(funcArr, result[0])
	}
}

func WriteWithIoutil(name, content string) {
	data := []byte(content)
	if ioutil.WriteFile(name, data, 0644) == nil {
		fmt.Println("写入文件成功")
	}
}
func combine() string {
	content := `package ` + pack + `

import (
	"context"
)

type (
	` + service + ` struct {
	}
)

func New` + service + `() (*` + service + `, error) {
	return &` + service + `{}, nil
}

`
	for _, value := range funcArr {
		funcStr := `func (h *` + service + `) ` + value[1] + `(_ context.Context, req *` + value[2] + `) (*` + value[3] + `, error) {
	resp := new(` + value[3] + `)
	return resp, nil
}

`
		content += funcStr
	}
	return content
}
