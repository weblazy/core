package timex

import "time"

const (
	TimeFormat = "2006-01-02 15:04:05"
	DateFormat = "2006-01-02"
)

func NowDateStr() string {
	return time.Now().Format(DateFormat)
}

func DateStr(t time.Time) string {
	return t.Format(DateFormat)
}

func NowTimeStr() string {
	return time.Now().Format(TimeFormat)
}

func TimeStr(t time.Time) string {
	return t.Format(TimeFormat)
}

// Use the long enough past time as start time, in case timex.Now() - lastTime equals 0.

func Now() time.Duration {
	return time.Duration(time.Now().UnixNano())
}

func Since(d time.Duration) time.Duration {
	return Now() - d
}

func Time(d time.Duration) time.Time {
	return time.Unix(0, int64(d))
}
