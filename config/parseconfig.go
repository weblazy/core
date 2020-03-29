package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"lazygo/core/database/redis"
	"lazygo/core/rpcx"
)

type (
	Configure struct {
		Env          string             //dev,pre,pro环境变量
		ConfigureRpc rpcx.RpcClientConf //配置中心Rpc
		Username     string
		Password     string
		Network      string //outside外网,inside内网,解决外网内网连接不一致的情况
	}
)

const (
	redisType = "redis"
	mysqlType = "mysql"
	mongoType = "mongo"
	rpcType   = "rpc"
	arrType   = "arr"
)

var (
	env = flag.String("e", "", "The env file")
	// configureModel *configure.ConfigureModel
)

func SetConfigureModel(configString string, config interface{}) error {
	var err error
	if *env != "" {
		var c Configure
		// MustLoad(*env, &c) //加载
		// configureModel = configure.NewConfigureModel(rpcx.MustNewClient(c.ConfigureRpc), c.Username, c.Password, c.Env, c.Network)
		time.Sleep(time.Duration(1) * time.Second)
		err = parseConfig(config, c.Env)
		if err != nil {
			panic(err)
		}
		name, err := parseName(config)
		if err != nil {
			panic(err)
		}
		err = write(name, config)
		os.Exit(0)
	} else {
		data, err := readFile(configString)
		if err != nil {
			return err
		}
		json.Unmarshal(data, config)
	}
	return err
}

func readFile(path string) ([]byte, error) {
	fi, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fi.Close()
	fd, err := ioutil.ReadAll(fi)
	if err != nil {
		return nil, err
	}
	return fd, nil
}

func write(name string, config interface{}) error {
	fd, err := os.OpenFile(name+".json", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	buf := []byte(data)
	fd.Write(buf)
	return nil
}

func parseName(ptr interface{}) (string, error) {
	t := reflect.TypeOf(ptr)
	// 入参类型校验
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return "", fmt.Errorf("The ptr argument must be a structure pointer")
	}
	// 取指针指向的结构体变量
	v := reflect.ValueOf(ptr).Elem()
	fieldInfo, ok := v.Type().FieldByName("Name")
	if !ok {
		return "", fmt.Errorf("No field named Name")
	}
	tag := fieldInfo.Tag
	name := tag.Get("all")
	if name == "" {
		return "", fmt.Errorf("Name field No Tag named all")
	}
	// resp, err := configureModel.Server(name)
	// if resp == nil || err != nil {
	// 	return "", fmt.Errorf("No Server named %s", name)
	// }

	// if resp.ServerType == "api" {
	// 	v.FieldByName("Host").SetString("0.0.0.0")
	// 	v.FieldByName("Port").SetInt(resp.Port)
	// } else if resp.ServerType == "rpc" {
	// 	v.FieldByName("ListenOn").SetString(fmt.Sprintf("0.0.0.0:%d", resp.Port))
	// }
	return name, nil
}

func parseConfig(ptr interface{}, env string) error {
	if env != "dev" && env != "pre" && env != "pro" {
		return fmt.Errorf("The env argument must be in dev,pre,pro")
	}
	t := reflect.TypeOf(ptr)
	v := reflect.ValueOf(ptr).Elem()
	// 入参类型校验
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("The ptr argument must be a structure pointer")
	}
	count := v.NumField()
	for i := 0; i < count; i++ {
		field := v.Field(i)
		fieldInfo := v.Type().Field(i)
		name := fieldInfo.Name
		tag := fieldInfo.Tag
		value := tag.Get("all")
		if value == "" {
			value = tag.Get(env)
		}
		kind := fieldInfo.Type.Kind()
		err := setValue(kind, field, value, env, name)
		if err != nil {
			return err
		}

	}
	return nil
}

func setValue(kind reflect.Kind, field reflect.Value, value, env, name string) error {
	var err error
	switch kind {
	case reflect.Bool:
		res, _ := strconv.ParseBool(value)
		field.Set(reflect.ValueOf(res))
	case reflect.Int:
		res, _ := strconv.ParseInt(value, 10, 0)
		field.Set(reflect.ValueOf(int(res)))
	case reflect.Int8:
		res, _ := strconv.ParseInt(value, 10, 8)
		field.Set(reflect.ValueOf(int8(res)))
	case reflect.Int16:
		res, _ := strconv.ParseInt(value, 10, 16)
		field.Set(reflect.ValueOf(int16(res)))
	case reflect.Int32:
		res, _ := strconv.ParseInt(value, 10, 32)
		field.Set(reflect.ValueOf(int32(res)))
	case reflect.Int64:
		res, _ := strconv.ParseInt(value, 10, 64)
		field.Set(reflect.ValueOf(res))
	case reflect.Uint:
		res, _ := strconv.ParseUint(value, 10, 0)
		field.Set(reflect.ValueOf(uint(res)))
	case reflect.Uint8:
		res, _ := strconv.ParseUint(value, 10, 8)
		field.Set(reflect.ValueOf(uint8(res)))
	case reflect.Uint16:
		res, _ := strconv.ParseUint(value, 10, 16)
		field.Set(reflect.ValueOf(uint16(res)))
	case reflect.Uint32:
		res, _ := strconv.ParseUint(value, 10, 32)
		field.Set(reflect.ValueOf(uint32(res)))
	case reflect.Uint64:
		res, _ := strconv.ParseUint(value, 10, 64)
		field.Set(reflect.ValueOf(res))
	case reflect.Float32:
		res, _ := strconv.ParseFloat(value, 32)
		field.Set(reflect.ValueOf(float32(res)))
	case reflect.Float64:
		res, _ := strconv.ParseFloat(value, 64)
		field.Set(reflect.ValueOf(res))
	case reflect.String:
		// r := regexp.MustCompile("{{(.*)}}")
		// result := r.FindAllStringSubmatch(value, -1)
		// if len(result) > 0 {
		// 	resp, err := configureModel.Source(result[0][1])
		// 	if resp == nil || err != nil {
		// 		return err
		// 	}
		// 	value = resp.Source
		// }
		field.Set(reflect.ValueOf(value))
	case reflect.Struct:
		pointer := field.Addr().Interface()
		if value != "" {
			r := regexp.MustCompile("{{(.*)}}")
			result := r.FindAllStringSubmatch(value, -1)
			if len(result) > 0 {
				resp, err := parseStruct(pointer, result[0][1])
				if err != nil {
					return err
				}
				field.Set(resp)
			}
		} else {
			err = parseConfig(pointer, env)
			if err != nil {
				return err
			}
		}
	default:
		if value != "" {
			return fmt.Errorf("%s field type is not supported, please use json string instead", name)
		}
	}
	return nil
}

func parseStruct(value interface{}, name string) (reflect.Value, error) {
	var result reflect.Value
	switch value.(type) {
	// case *baseconst.AuthConfig:
	// resp, err := configureModel.Source(name)
	// if err != nil {
	// 	return result, err
	// }
	// if resp == nil {
	// 	return result, fmt.Errorf("%s name is not found", name)
	// }
	// var auth baseconst.AuthConfig
	// json.Unmarshal([]byte(resp.Source), &auth)
	// result = reflect.ValueOf(auth)
	// return result, nil
	case *rpcx.RpcClientConf:
		// resp, err := configureModel.Rpc(name)
		// if err != nil {
		// 	return result, err
		// }
		// if resp == nil {
		// 	return result, fmt.Errorf("%s name is not found", name)
		// }
		// result = reflect.ValueOf(rpcx.RpcClientConf{
		// 	Server: resp.Source,
		// 	App:    "app",
		// 	Token:  "token",
		// })
		// return result, nil
	case *redis.RedisConf:
		// resp, err := configureModel.Source(name)
		// if err != nil {
		// 	return result, err
		// }
		// if resp == nil {
		// 	return result, fmt.Errorf("%s name is not found", name)
		// }
		var redis redis.RedisConf
		// json.Unmarshal([]byte(resp.Source), &redis)
		result = reflect.ValueOf(redis)
		return result, nil
	}
	return result, fmt.Errorf("%s type is not found", name)
}
