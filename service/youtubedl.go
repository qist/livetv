package service

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/qist/livetv/global"
)

func GetYoutubeLiveM3U8(youtubeURL string) (string, error) {
	liveURL, ok := global.URLCache.Load(youtubeURL)
	if ok {
		return liveURL.(string), nil
	} else {
		log.Println("cache miss", youtubeURL)
		liveURL, err := RealGetYoutubeLiveM3U8(youtubeURL)
		if err != nil {
			log.Println(err)
			log.Println("[YTDL]", liveURL)
			return "", err
		} else {
			global.URLCache.Store(youtubeURL, liveURL)
			return liveURL, nil
		}
	}
}

func RealGetYoutubeLiveM3U8(youtubeURL string) (string, error) {
	YtdlCmd, err := GetConfig("ytdl_cmd")
	if err != nil {
		log.Println(err)
		return "", err
	}
	YtdlArgs, err := GetConfig("ytdl_args")
	if err != nil {
		log.Println(err)
		return "", err
	}
	YtdlCookies, err := GetConfig("ytdl_cookies")
	if err != nil {
		log.Println(err)
		return "", err
	}
	ytdlArgs := splitArgs(YtdlArgs)
	for i, v := range ytdlArgs {
		if strings.EqualFold(v, "{url}") {
			ytdlArgs[i] = youtubeURL
		}
	}
	ytdlCookies := strings.TrimSpace(YtdlCookies)
	if ytdlCookies != "" {
		if !filepath.IsAbs(ytdlCookies) {
			datadir := os.Getenv("LIVETV_DATADIR")
			if datadir == "" {
				datadir = "./data"
			}
			ytdlCookies = filepath.Join(datadir, ytdlCookies)
		}
		ytdlArgs = append(ytdlArgs, "--cookies", ytdlCookies)
	}
	_, err = exec.LookPath(YtdlCmd)
	if err != nil {
		log.Println(err)
		return "", err
	} else {
		timeout := global.HttpClientTimeout
		if cfgTimeout, err := GetConfig("ytdl_timeout"); err == nil {
			if sec, err := strconv.Atoi(cfgTimeout); err == nil && sec > 0 {
				timeout = time.Duration(sec) * time.Second
			}
		}
		ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
		defer cancelFunc()
		log.Println(YtdlCmd, ytdlArgs)
		cmd := exec.CommandContext(ctx, YtdlCmd, ytdlArgs...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Println(err)
			return "", err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Println(err)
			return "", err
		}
		if err := cmd.Start(); err != nil {
			log.Println(err)
			return "", err
		}
		var stdoutBytes []byte
		var stderrBytes []byte
		stdoutCh := make(chan error, 1)
		stderrCh := make(chan error, 1)
		go func() {
			b, readErr := io.ReadAll(stdout)
			stdoutBytes = b
			stdoutCh <- readErr
		}()
		go func() {
			b, readErr := io.ReadAll(stderr)
			stderrBytes = b
			stderrCh <- readErr
		}()

		waitErr := cmd.Wait()
		if err := <-stdoutCh; err != nil {
			log.Println(err)
			return "", err
		}
		if err := <-stderrCh; err != nil {
			log.Println(err)
			return "", err
		}
		if ctx.Err() == context.DeadlineExceeded {
			if len(stderrBytes) > 0 {
				log.Println("[YTDL stderr]", string(stderrBytes))
			}
			log.Println("[YTDL] timeout after", timeout)
			return "", ctx.Err()
		}
		if waitErr != nil {
			if len(stderrBytes) > 0 {
				log.Println("[YTDL stderr]", string(stderrBytes))
			}
			log.Println(waitErr)
			return "", waitErr
		}
		if len(stderrBytes) > 0 {
			log.Println("[YTDL stderr]", string(stderrBytes))
		}
		return strings.TrimSpace(string(stdoutBytes)), nil
	}
}

// splitArgs parses a command-line string into arguments with shell-like quoting.
// It supports single quotes, double quotes, and backslash escapes.
func splitArgs(s string) []string {
	var args []string
	var buf strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	flush := func() {
		if buf.Len() > 0 {
			args = append(args, buf.String())
			buf.Reset()
		}
	}

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			buf.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' && !inSingle {
			escaped = true
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if (ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r') && !inSingle && !inDouble {
			flush()
			continue
		}
		buf.WriteByte(ch)
	}
	flush()
	return args
}
