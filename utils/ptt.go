package utils

import (
	"regexp"
	"strconv"
)

const pttTable = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_"

var pttReg = regexp.MustCompile(`M.(\d{10}).A.([\dA-F]{3}).html`)

func Url2Aid(url string) string {
	var aid string
	matches := pttReg.FindStringSubmatch(url)
	if len(matches) == 0 {
		return aid
	}

	ts, _ := strconv.Atoi(matches[1])
	var tsAid string
	for ts > 0 {
		remainder := ts % 64
		tsAid = string(pttTable[remainder]) + tsAid
		ts /= 64
	}

	rd, _ := strconv.ParseInt(matches[2], 16, 64)
	var rdAid string
	for rd > 0 {
		remainder := rd % 64
		rdAid = string(pttTable[remainder]) + rdAid
		rd /= 64
	}

	aid = tsAid + rdAid

	return aid
}

func Aid2Url(aid string) string {
	return ""
}
