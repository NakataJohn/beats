package common

import (
	"regexp"
	"testing"
)

func TestJson(t *testing.T) {

	str := `
	{
		"action":"start",
		"data": [{"id":""}]
	}
	`
	reg := regexp.MustCompile("")
	reg.Find([]byte(str))

}
