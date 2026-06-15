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
		zw, _ := gzip.NewWriterLevel(nil, gzip.DefaultCompression)
		return zw
	},
}

var gzipReaderPool = sync.Pool{
	New: func() any {
		return new(gzip.Reader)
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
	zr := gzipReaderPool.Get().(*gzip.Reader)
	if err := zr.Reset(bytes.NewReader(zipData)); err != nil {
		gzipReaderPool.Put(zr)
		return "", err
	}
	result, err := io.ReadAll(zr)
	zr.Close()
	gzipReaderPool.Put(zr)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
