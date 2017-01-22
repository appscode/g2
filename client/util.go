package client

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type bounds struct {
	low, high int
	t         string
}

var (
	Minute = bounds{0, 59, "minute"}
	Hour   = bounds{0, 24, "hour"}
	Day    = bounds{1, 31, "day"}
	Month  = bounds{1, 12, "month"}
	Year   = bounds{1970, 2200, "year"}
)

func ToEpoch(exp string) (int64, error) {
	f := strings.Fields(exp)
	if len(f) != 5 {
		return 0, errors.New("expected 5 field")
	}
	mi, err := validateAndGet(f[0], Minute)
	if err != nil {
		return 0, err
	}
	hi, err := validateAndGet(f[1], Hour)
	if err != nil {
		return 0, err
	}
	di, err := validateAndGet(f[2], Day)
	if err != nil {
		return 0, err
	}
	moi, err := validateAndGet(f[3], Month)
	if err != nil {
		return 0, err
	}
	yi, err := validateAndGet(f[4], Year)
	if err != nil {
		return 0, err
	}
	t := time.Date(yi, time.Month(moi), di, hi, mi, 0, 0, time.Local)
	return t.UTC().Unix(), nil
}

func validateAndGet(v string, b bounds) (int, error) {
	num, err := strconv.Atoi(v)
	if err != nil {
		return num, err
	}
	if num < b.low || num > b.high {
		return num, fmt.Errorf("%v should be between %v and %v", b.t, b.low, b.high)
	}
	return num, nil
}
