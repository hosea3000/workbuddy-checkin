package main

import (
	"net/url"
	"strings"
	"time"
)

// timeNow 便于测试注入的可替换时钟。
var timeNow = time.Now

// safeExternalURL 校验外部 URL：无控制字符、绝对 http(s)、有 host、无 userinfo。
func safeExternalURL(v string) bool {
	if v == "" || v != strings.TrimSpace(v) {
		return false
	}
	for _, r := range v {
		if r < 32 || r == 127 {
			return false
		}
	}
	u, err := url.Parse(v)
	if err != nil {
		return false
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return false
	}
	return true
}
