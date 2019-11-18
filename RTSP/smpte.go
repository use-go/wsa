package RTSP

import "strings"

//https://blog.csdn.net/andrew57/article/details/6752182
/*
smpte=10:12:33:20-
smpte=10:07:33-
smpte=10:07:00-10:07:33:05.01
smpte-25=10:07:00-10:07:33:05.01
*/
const SMPTE_30_drop_frame_rate = 29.97

const smpte_prefix = "smpte"

func IsSMPTE(line string) bool {
	if !strings.HasPrefix(line, smpte_prefix) {
		return false
	}

	if strings.Count(line, "=") != 1 || strings.Count(line, "-") != 1 {
		return false
	}

	eqIndex := strings.Index(line, "=")
	hyphenIndex := strings.Index(line, "-")

	if eqIndex == -1 || hyphenIndex == -1 || eqIndex > hyphenIndex {
		return false
	}

	return true
}

func 
