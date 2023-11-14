package main

import (
	"fmt"
	"strings"
)

func is_duplicate_entry(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "Duplicate entry") {
		return true
	}
	return false
}

var ERROR_NOT_SELLER error = fmt.Errorf("不存在商家链接")
