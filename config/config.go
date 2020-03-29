package config

import (
	"encoding/json"
	"lazygo/core/fs"
	"lazygo/core/logx"
)

const (
	VERSION = "1.9.2"
)

type ConfigInterface interface {
	MustSetUp()
}

type Config struct {
	Log        logx.Config
	AppName    string //Application name
	RunMode    string //Running Mode: dev | prod
	ServerName string
}

type ApiConfig struct {
	Config
	Port      int64
	MaxMemory int64
	Timeout   int64
}

type RpcConfig struct {
	Config
	Port int64
}

func UnmarshalWithLog(configFile string, config interface{}) {
	data, err := fs.ReadFile(configFile)
	if err != nil {
		logx.Fatal(err)
	}
	json.Unmarshal(data, config)
	c := config.(ConfigInterface)
	c.MustSetUp()
}

func Unmarshal(configFile string, config interface{}) {
	data, err := fs.ReadFile(configFile)
	if err != nil {
		logx.Fatal(err)
	}
	json.Unmarshal(data, config)
}

func (c Config) MustSetUp() {
	if err := c.SetUp(); err != nil {
		logx.Fatal(err)
	}
}

func (c Config) SetUp() error {
	if err := logx.SetUp(c.Log); err != nil {
		return err
	}
	return nil
}
