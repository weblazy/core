package apix

import (
	"fmt"
)

func Pm(m map[string]string) {
	for k, v := range m {
		fmt.Println(k, v)
	}
}

func Ps(s []string) {
	for k, v := range s {
		fmt.Println(k, v)
	}
}
