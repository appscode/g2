package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToEpoch(t *testing.T) {
	testData := []struct {
		exp      string
		val      int64
		hasError bool
	}{
		{"* * * * *", 0, true},
		{"28 9 10 1 2017", 1484018880, false},
		{"28 10 1 2017", 0, true},
		{"28 9 10 1 1970", 790080, false},
	}
	for _, td := range testData {
		num, err := ToEpoch(td.exp)
		if td.hasError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, td.val, num)
		}
	}
}

func TestValidateAndGet(t *testing.T) {
	testData := []struct {
		tc       int
		str      string
		b        bounds
		val      int
		hasError bool
	}{
		{1, "1970", Year, 1970, false},
		{2, "2020", Year, 2020, false},
		{3, "2200", Year, 2200, false},
		{4, "1", Month, 1, false},
		{5, "5", Month, 5, false},
		{6, "12", Month, 12, false},
		{7, "1", Day, 1, false},
		{8, "15", Day, 15, false},
		{9, "31", Day, 31, false},
		{10, "0", Hour, 0, false},
		{11, "12", Hour, 12, false},
		{12, "23", Hour, 23, false},
		{13, "0", Minute, 0, false},
		{14, "30", Minute, 30, false},
		{15, "59", Minute, 59, false},
		{16, "1900", Year, 1900, true},
		{17, "13", Month, 13, true},
		{18, "32", Day, 32, true},
		{19, "-1", Hour, -1, true},
		{20, "66", Minute, 66, true},
	}
	for _, td := range testData {
		num, err := validateAndGet(td.str, td.b)
		if td.hasError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, td.val, num)
		}
	}
}
