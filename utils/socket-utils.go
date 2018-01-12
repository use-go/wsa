// Copyright 2017-2018 The use-go websocket-streamserver Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package utils

import (
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/use-go/websocket-streamserver/logger"
)

// TCPRead to get data from tcp buffer
func TCPRead(conn net.Conn, size int) (data []byte, err error) {
	data = make([]byte, size)
	received := 0
	for received < int(size) {
		ret, err := conn.Read(data[received:])
		if err != nil {
			logger.LOGE(err.Error())
			return data[:received], err
		}
		received += ret
	}
	return data, err
}

// TCPReadTimeout control
func TCPReadTimeout(conn net.Conn, size int, millSec int) (data []byte, err error) {
	if millSec > 0 {
		err = conn.SetReadDeadline(time.Now().Add(time.Duration(millSec) * time.Millisecond))
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
		defer func() {
			conn.SetReadDeadline(time.Time{})
		}()
	}
	data = make([]byte, size)
	received := 0
	for received < int(size) {
		ret, err := conn.Read(data[received:])
		if err != nil {
			logger.LOGE(err.Error())
			return data[:received], err
		}
		received += ret
	}
	return data, err
}

// TCPReadTimeDuration timeout
func TCPReadTimeDuration(conn net.Conn, size int, duration time.Duration) (data []byte, err error) {
	if duration > 0 {
		err = conn.SetReadDeadline(time.Now().Add(duration))
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
		defer func() {
			conn.SetReadDeadline(time.Time{})
		}()
	}
	data = make([]byte, size)
	received := 0
	for received < int(size) {
		ret, err := conn.Read(data[received:])
		if err != nil {
			logger.LOGE(err.Error())
			return data[:received], err
		}
		received += ret
	}
	return data, err
}

// TCPWrite data
func TCPWrite(conn net.Conn, data []byte) (writed int, err error) {
	err = conn.SetReadDeadline(time.Now().Add(time.Hour))
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	defer func() {
		conn.SetReadDeadline(time.Time{})
	}()
	for writed < len(data) {
		ret, err := conn.Write(data[writed:])
		if err != nil {
			//logger.LOGE(err.Error())
			return writed, err
		}
		writed += ret
	}
	return
}

//TCPWriteTimeOut control
func TCPWriteTimeOut(conn net.Conn, data []byte, millSec int) (writed int, err error) {
	err = conn.SetWriteDeadline(time.Now().Add(time.Millisecond * time.Duration(millSec)))
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	defer func() {
		conn.SetWriteDeadline(time.Time{})
	}()
	for writed < len(data) {
		ret, err := conn.Write(data[writed:])
		if err != nil {
			logger.LOGE(err.Error())
			return writed, err
		}
		writed += ret
	}
	return
}

// TCPWriteTimeDuration to control timeout
func TCPWriteTimeDuration(conn net.Conn, data []byte, duration time.Duration) (writed int, err error) {
	err = conn.SetWriteDeadline(time.Now().Add(duration))
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	defer func() {
		conn.SetWriteDeadline(time.Time{})
	}()
	for writed < len(data) {
		ret, err := conn.Write(data[writed:])
		if err != nil {
			logger.LOGE(err.Error())
			return writed, err
		}
		writed += ret
	}
	return
}

//IP2Int Enocde IP to a int
func IP2Int(ip string) int {
	num := 0
	ipSections := strings.Split(ip, ".")

	intIPSec1, _ := strconv.Atoi(ipSections[0])
	intIPSec1 *= 256 * 256 * 256
	intIPSec2, _ := strconv.Atoi(ipSections[1])
	intIPSec1 *= 256 * 256
	intIPSec3, _ := strconv.Atoi(ipSections[2])
	intIPSec1 *= 256
	intIPSec4, _ := strconv.Atoi(ipSections[3])
	num = intIPSec1 + intIPSec2 + intIPSec3 + intIPSec4
	num = num >> 0
	return num
}

//ConnCopy transfer data from one socket to another one
// This does the actual data transfer.
// The broker only closes the Read side.
func ConnCopy(src, dst net.Conn, reverseChan chan int64) (err error) {
	// We can handle errors in a finer-grained manner by inlining io.Copy (it's
	// simple, and we drop the ReaderFrom or WriterTo checks for
	// net.Conn->net.Conn transfers, which aren't needed). This would also let
	// us adjust buffersize.
	reverseBytes, err := io.Copy(dst, src)
	if err != nil {
		err = errors.New("Server closed connection")
	}

	// close all connections discarding errors
	// this is force the other thread to close
	src.Close()
	dst.Close()

	//clog.Printf("Wrote %d bytes to reverse connection\n", reverse_bytes)
	reverseChan <- reverseBytes

	return
}
