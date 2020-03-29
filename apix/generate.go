package apix

import (
	"fmt"
	"io/ioutil"
	"lazygo/conf"
	"os"
	"strings"
)

func getfiles(path string) {
	//读取模板文件
	b, err := ioutil.ReadFile(conf.TPL_PATH + "/router.tpl")
	if err != nil {
		panic(err)
	}
	router_str := string(b)
	//遍历控制器
	files, _ := ioutil.ReadDir(path)
	controllers_str := ""
	for _, file := range files {
		if file.IsDir() {
			continue
		} else {
			if strings.HasSuffix(file.Name(), "Controller.go") {
				filenameOnly := strings.TrimSuffix(file.Name(), ".go")
				key := strings.TrimSuffix(file.Name(), "Controller.go")
				controllers_str = controllers_str + tab(2) + "\"/" + key + "\" : &controllers." + filenameOnly + "{}," + enter(1)
			}
		}

	}
	router_str = strings.Replace(router_str, "[CONTROLLER_MAP]", controllers_str, -1)
	data := []byte(router_str)
	if ioutil.WriteFile(conf.CONF_PATH+"/router.go", data, 0644) == nil {
		fmt.Println("写入文件成功:", router_str)
	}
}

/**
 * 判断文件是否存在  存在返回 true 不存在返回false
 */
func isExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

/**
 * 模拟tab产生空格
 * @param int $step
 * @return string
 */
func tab(step int) string {
	var str string = ""
	for index := 0; index < step; index++ {
		str = str + "    "
	}
	return str
}

/**
 * 模拟enter产生空格
 * @param int $step
 * @return string
 */
func enter(step int) string {
	var str string = ""
	for index := 0; index < step; index++ {
		str = str + "\r\n"
	}
	return str
}

/**
 * 模拟enter产生空格
 * @param int $step
 * @return string
 */
func space(step int) string {
	var str string = ""
	for index := 0; index < step; index++ {
		str = str + ""
	}
	return str
}

func run() {

}
