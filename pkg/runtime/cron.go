package runtime

import (
	"fmt"

	"github.com/appscode/errors"
	"gopkg.in/robfig/cron.v2"
)

const starBit = 1 << 63

type CronSpecInterface interface {
	Bytes() []byte
	Expr() string
	Schedule() cron.Schedule
}

type cronSpec struct {
	specByte   []byte
	expression string
	schedule   *cron.SpecSchedule
}

func NewCronSchedule(expr string) (CronSpecInterface, error) {
	scd, err := cron.Parse(expr)
	if err != nil {
		return nil, err
	}
	specScd, ok := scd.(*cron.SpecSchedule)
	if !ok {
		return nil, errors.New("invalid cron expression")
	}
	cronByte := getBytes(specScd.Minute, specScd.Hour, specScd.Dom, specScd.Month, specScd.Dow)
	return cronSpec{
		specByte:   cronByte,
		expression: expr,
		schedule:   specScd,
	}, nil
}

func (c cronSpec) Schedule() cron.Schedule {
	return c.schedule
}

func (c cronSpec) Bytes() []byte {
	return c.specByte
}

func (c cronSpec) Expr() string {
	return c.expression
}

func getBytes(data ...uint64) []byte {
	var res []byte = make([]byte, 0)
	for _, val := range data {
		res = append(res, []byte(fmt.Sprintf("%v\x00", toStringOrEmptyForStar(val)))...)
	}
	return res
}

func toStringOrEmptyForStar(n uint64) string {
	if hasStar(n) {
		return ""
	}
	return fmt.Sprintf("%v", firstSetBitPos(n))
}

func firstSetBitPos(n uint64) uint64 {
	var i uint64
	for i = 0; i < 64; i++ {
		if ((1 << i) & n) != 0 {
			return i
		}
	}
	return i
}

func hasStar(n uint64) bool {
	return (n & starBit) != 0
}
