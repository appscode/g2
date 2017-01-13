package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCronSchedule(t *testing.T) {
	testData := []struct {
		Expr         string
		HasError     bool
		ExpectedByte []byte
	}{
		{
			Expr:         "0 1 2 3 4",
			HasError:     false,
			ExpectedByte: []byte{48, 0, 49, 0, 50, 0, 51, 0, 52, 0},
		},
		{
			Expr:         "* 1 2 3 4",
			HasError:     false,
			ExpectedByte: []byte{0, 49, 0, 50, 0, 51, 0, 52, 0},
		},
		{
			Expr:         "* * * * *",
			HasError:     false,
			ExpectedByte: []byte{0, 0, 0, 0, 0},
		},
		{
			Expr:         "* * * 0 *",
			HasError:     true,
			ExpectedByte: []byte{0, 0, 0, 0, 0},
		},
		{
			Expr:         "* * * * 0",
			HasError:     false,
			ExpectedByte: []byte{0, 0, 0, 0, 48, 0},
		},
		{
			Expr:         "* * * - *",
			HasError:     true,
			ExpectedByte: []byte{48, 0, 48, 0, 49, 0, 49, 0, 48, 0},
		},
	}
	for _, td := range testData {
		gotObj, gotErr := NewCronSchedule(td.Expr)
		if td.HasError {
			assert.NotNil(t, gotErr)
		} else {
			assert.Nil(t, gotErr)
			assert.Equal(t, td.ExpectedByte, gotObj.Bytes())
		}
	}

	_, err := NewCronSchedule("@every 1h")
	assert.NotNil(t, err)
}
