package stringx

import (
	"errors"
)

var (
	ErrInvalidStartPosition = errors.New("start position is invalid")
	ErrInvalidStopPosition  = errors.New("stop position is invalid")
)

func Contains(list []string, str string) bool {
	for _, each := range list {
		if each == str {
			return true
		}
	}

	return false
}

func HasEmpty(args ...string) bool {
	for _, arg := range args {
		if len(arg) == 0 {
			return true
		}
	}

	return false
}

func NotEmpty(args ...string) bool {
	return !HasEmpty(args...)
}

func Remove(strings []string, strs ...string) []string {
	out := append([]string(nil), strings...)

	for _, str := range strs {
		var n int
		for _, v := range out {
			if v != str {
				out[n] = v
				n++
			}
		}
		out = out[:n]
	}

	return out
}

func Reverse(s string) string {
	runes := []rune(s)

	for from, to := 0, len(runes)-1; from < to; from, to = from+1, to-1 {
		runes[from], runes[to] = runes[to], runes[from]
	}

	return string(runes)
}

// Substr returns runes between start and stop [start, stop) regardless of the chars are ascii or utf8
func Substr(str string, start int, stop int) (string, error) {
	rs := []rune(str)
	length := len(rs)

	if start < 0 || start > length {
		return "", ErrInvalidStartPosition
	}

	if stop < 0 || stop > length {
		return "", ErrInvalidStopPosition
	}

	return string(rs[start:stop]), nil
}

func TakeOne(valid, or string) string {
	if len(valid) > 0 {
		return valid
	} else {
		return or
	}
}

func Union(first, second []string) []string {
	set := make(map[string]struct{})

	for _, each := range first {
		set[each] = struct{}{}
	}
	for _, each := range second {
		set[each] = struct{}{}
	}

	merged := make([]string, 0, len(set))
	for k := range set {
		merged = append(merged, k)
	}

	return merged
}
