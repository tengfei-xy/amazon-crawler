package main

import (
	"net"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

func get_socks5_proxy() (proxy.Dialer, error) {
	// 创建一个SOCKS5代理拨号器
	return proxy.SOCKS5("tcp", app.Proxy.Sockc5[0], nil, proxy.Direct)
}
func get_client() http.Client {

	proxy, err := get_socks5_proxy()
	if err != nil {
		return http.Client{Timeout: time.Second * 60}
	}
	return http.Client{
		Transport: &http.Transport{
			Dial: proxy.Dial,
		},

		Timeout: time.Second * 60,
	}
}

func telnet(ip string) bool {
	conn, err := net.DialTimeout("tcp", ip, 5*time.Second)
	if err != nil {
		return false
	} else {
		if conn != nil {
			_ = conn.Close()
			return true
		} else {
			return false
		}
	}
}
