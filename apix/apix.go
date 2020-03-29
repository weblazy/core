package apix

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"lazygo/conf"
	"lazygo/core/apix/httphandler"
	"lazygo/core/config"
	"lazygo/core/logx"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (

	// HTTPMETHOD list the supported http methods.
	HTTPMETHOD = map[string]bool{
		"GET":       true,
		"POST":      true,
		"PUT":       true,
		"DELETE":    true,
		"PATCH":     true,
		"OPTIONS":   true,
		"HEAD":      true,
		"TRACE":     true,
		"CONNECT":   true,
		"MKCOL":     true,
		"COPY":      true,
		"MOVE":      true,
		"PROPFIND":  true,
		"PROPPATCH": true,
		"LOCK":      true,
		"UNLOCK":    true,
	}
	forbidMethod      map[string]bool            = make(map[string]bool, 0)
	controllerInfoMap map[string]*ControllerInfo = make(map[string]*ControllerInfo, 0)
	ApiConfig         config.ApiConfig
)

type (
	// ControllerRegister containers registered router rules, controller handlers and filters.
	ControllerRegister struct {
		// routers      map[string]*Tree
		enablePolicy bool
		// policies     map[string]*Tree
		enableFilter bool
		// filters      [FinishRouter + 1][]*FilterRouter
		pool sync.Pool
	}

	ControllerInfo struct {
		key            string
		keyLenth       int
		controllerType reflect.Type
		methodMap      map[string]bool
	}
)

// NewControllerRegister returns a new ControllerRegister.
func NewControllerRegister() *ControllerRegister {
	cr := &ControllerRegister{
		// routers:  make(map[string]*Tree),
		// policies: make(map[string]*Tree),
	}
	return cr
}

func Run(conf config.ApiConfig, routerMap map[string]ControllerInterface) {
	mux := NewControllerRegister()
	initForbidMethod()
	for key, value := range routerMap {
		vf := reflect.ValueOf(value)
		vft := vf.Type()
		vtp := reflect.Indirect(vf).Type()

		//读取方法数量
		mNum := vf.NumMethod()
		//遍历路由器的方法，并将其存入控制器映射变量中
		methodMap := make(map[string]bool, 0)
		for i := 0; i < mNum; i++ {
			mName := vft.Method(i).Name
			if forbid, ok := forbidMethod[mName]; ok {
				methodMap[mName] = forbid
			} else {
				methodMap[mName] = true
			}
		}
		controllerInfoMap[key] = &ControllerInfo{
			key:            key,
			keyLenth:       len(key),
			controllerType: vtp,
			methodMap:      methodMap,
		}
	}

	ApiConfig = conf
	if ApiConfig.MaxMemory == 0 {
		ApiConfig.MaxMemory = 1 << 26 //64M
	}
	timeoutHandler := httphandler.TimeoutHandler(time.Duration(ApiConfig.Timeout) * time.Millisecond)
	hander := timeoutHandler(mux)
	portStr := strconv.FormatInt(ApiConfig.Port, 10)
	logx.Info("http server Running on http://:" + portStr)
	http.ListenAndServe(":"+portStr, hander)
}

func initForbidMethod() {
	vf := reflect.ValueOf(new(Controller))
	vft := vf.Type()
	//读取方法数量
	mNum := vf.NumMethod()
	//遍历路由器的方法，并将其存入控制器映射变量中
	for i := 0; i < mNum; i++ {
		mName := vft.Method(i).Name
		forbidMethod[mName] = false
	}
}

func (p *ControllerRegister) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	var (
		urlPath        = r.URL.Path
		controllerInfo *ControllerInfo
		execController ControllerInterface
	)
	// filter wrong http method
	if !HTTPMETHOD[r.Method] {
		http.Error(w, "Method Not Allowed", 405)
		goto Admin
	}

	if urlPath == "/favicon.ico" || urlPath == "/robots.txt" {
		file := conf.IMG_PATH + "/favicon.ico"
		f, err := os.Open(file)
		defer f.Close()
		if err != nil && os.IsNotExist(err) {
			file = conf.IMG_PATH + "/default.png"
		}
		http.ServeFile(w, r, file)
		return
	}

	for key, value := range controllerInfoMap {
		if strings.HasPrefix(urlPath, key) {
			if controllerInfo == nil || (value.keyLenth > controllerInfo.keyLenth) {
				controllerInfo = value
			}
		}
	}

	if controllerInfo != nil {
		methodName := urlPath[controllerInfo.keyLenth:]
		vc := reflect.New(controllerInfoMap[controllerInfo.key].controllerType)
		var ok bool
		execController, ok = vc.Interface().(ControllerInterface)
		if !ok {
			logx.Fatal("controller is not ControllerInterface")
		}
		baseController := BaseController{
			controllerName: controllerInfo.controllerType.Name(),
			actionName:     methodName,
			AppController:  execController,
			W:              w,
			R:              r,
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
				baseController.RequestBody = copyBody(w, r, ApiConfig.MaxMemory)
			}
			parseFormOrMulitForm(r, ApiConfig.MaxMemory)
		}

		execController.Init(baseController)
		if runable, ok := controllerInfo.methodMap[methodName]; runable && ok {
			method := vc.MethodByName(methodName)
			method.Call(nil)
		} else {
			http.NotFound(w, r)
		}
		goto Admin
	}
	http.NotFound(w, r)
Admin:
	//admin module record QPS
	record := map[string]interface{}{
		// "RemoteAddr":     context.Input.IP(),
		"RequestTime":    startTime,
		"DuringTime":     time.Since(startTime),
		"RequestMethod":  r.Method,
		"Request":        fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto),
		"ServerProtocol": r.Proto,
		"Host":           r.Host,
		// "Status":         statusCode,
		"HTTPReferrer":  r.Header.Get("Referer"),
		"HTTPUserAgent": r.Header.Get("User-Agent"),
		"RemoteUser":    r.Header.Get("Remote-User"),
		"BodyBytesSent": 0, //@todo this one is missing!
	}
	logx.Info(record)
	return
}

// CopyBody returns the raw request body data as bytes.
func copyBody(w http.ResponseWriter, r *http.Request, MaxMemory int64) []byte {
	if r.Body == nil {
		return []byte{}
	}

	var requestBody []byte
	safe := &io.LimitedReader{R: r.Body, N: MaxMemory}
	if r.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(safe)
		if err != nil {
			return nil
		}
		requestBody, _ = ioutil.ReadAll(reader)
	} else {
		requestBody, _ = ioutil.ReadAll(safe)
	}

	r.Body.Close()
	bf := bytes.NewBuffer(requestBody)
	r.Body = http.MaxBytesReader(w, ioutil.NopCloser(bf), MaxMemory)
	return requestBody
}

// ParseFormOrMulitForm parseForm or parseMultiForm based on Content-type
func parseFormOrMulitForm(r *http.Request, maxMemory int64) error {
	// Parse the body depending on the content type.
	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(maxMemory); err != nil {
			return errors.New("Error parsing request body:" + err.Error())
		}
	} else if err := r.ParseForm(); err != nil {
		return errors.New("Error parsing request body:" + err.Error())
	}
	return nil
}
