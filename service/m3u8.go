package service

import (
	"bufio"
	"strings"

	"github.com/qist/livetv/util"
)

func M3U8Process(data string, prefixURL string) string {
	var sb strings.Builder
	sb.Grow(len(data) + strings.Count(data, "\n")*len(prefixURL))
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, "#") {
			sb.WriteString(l)
		} else {
			sb.WriteString(prefixURL)
			sb.WriteString(util.CompressString(l))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
