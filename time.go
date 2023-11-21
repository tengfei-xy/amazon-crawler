package main

import (
	"time"

	log "github.com/tengfei-xy/go-log"
)

// 随机挂起 x 秒
func sleep(i int) {
	log.Infof("挂起%d秒", i)
	time.Sleep(time.Duration(i) * time.Second)
}
