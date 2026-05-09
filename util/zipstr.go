package util

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"sync"
)

var gzipWriterPool = sync.Pool{
	New: func() any {
		zw, _ := gzip.NewWriterLevel(nil, gzip.BestCompression)
		return zw
	},
}

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func CompressString(s string) string {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	zw := gzipWriterPool.Get().(*gzip.Writer)
	zw.Reset(buf)
	_, _ = zw.Write([]byte(s))
	_ = zw.Close()
	zipResult := make([]byte, buf.Len())
	copy(zipResult, buf.Bytes())
	gzipWriterPool.Put(zw)
	bufferPool.Put(buf)
	return base64.URLEncoding.EncodeToString(zipResult)
}

func DecompressString(s string) (string, error) {
	zipData, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	zipReader, err := gzip.NewReader(bytes.NewBuffer(zipData))
	if err != nil {
		return "", err
	}
	result, err := io.ReadAll(zipReader)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
