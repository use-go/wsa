// Copyright 2017-2018 The use-go websocket-streamserver Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.


package utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"github.com/use-go/websocket-streamserver/logger"
)

//CheckDirectory in FS
func CheckDirectory(dir string) (bool, error) {
	_, err := os.Stat(dir)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, err
	}
	return false, err
}

// CreateDirectory for log
func CreateDirectory(dir string) (bool, error) {
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		logger.LOGE(err.Error())
		return false, err
	}
	return true, nil
}

// ReadFileAll from json file
func ReadFileAll(filename string) (data []byte, err error) {
	fp, err := os.OpenFile(filename, os.O_RDONLY, 0666)
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	defer fp.Close()
	data, err = ioutil.ReadAll(fp)
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	return
}


//GenerateGUID to Get a GUID
func GenerateGUID() string {
	b := make([]byte, 48)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return getMd5String(base64.URLEncoding.EncodeToString(b))
}

func getMd5String(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// InterfaceIsNil Check
func InterfaceIsNil(val interface{}) bool {
	if nil == val {
		return true
	}
	return reflect.ValueOf(val).IsNil()
}

// InterfaceValid Check
func InterfaceValid(val interface{}) bool {
	if nil == val {
		return false
	}
	return !reflect.ValueOf(val).IsNil()
}

// GetMD5Hash return the Encoded HASH
func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}