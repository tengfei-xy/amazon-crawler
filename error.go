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

var ERROR_NOT_SELLER_URL error = fmt.Errorf("不存在商家链接")
var ERROR_NOT_503 error = fmt.Errorf("连接失败,503")
var ERROR_NOT_404 error = fmt.Errorf("连接失败,404")
var ERROR_VERIFICATION error = fmt.Errorf("连接失败,需要验证")
