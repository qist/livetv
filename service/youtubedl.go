package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/qist/livetv/global"
)

type ytdlCall struct {
	done    chan struct{}
	liveURL string
	err     error
}

type ytdlFailure struct {
	err   error
	until time.Time
}

var (
	ytdlCallMu        sync.Mutex
	ytdlInflightCalls = make(map[string]*ytdlCall)
	ytdlFailures      = make(map[string]ytdlFailure)
)

const ytdlFailureBackoff = time.Minute

func GetYoutubeLiveM3U8(youtubeURL string) (string, error) {
	normalizedURL := normalizeYoutubeURL(youtubeURL)
	if normalizedURL == "" {
		return "", fmt.Errorf("empty youtube url")
	}
	if liveURL, ok := global.URLCache.Load(normalizedURL); ok {
		return liveURL.(string), nil
	}
	if cachedErr := getYtdlFailure(normalizedURL); cachedErr != nil {
		log.Println("skip yt-dlp retry during backoff", normalizedURL)
		return "", cachedErr
	}

	call, waiting := getOrCreateYtdlCall(normalizedURL)
	if waiting {
		<-call.done
		return call.liveURL, call.err
	}
	defer finishYtdlCall(normalizedURL, call)

	if liveURL, ok := global.URLCache.Load(normalizedURL); ok {
		call.liveURL = liveURL.(string)
		return call.liveURL, nil
	}

	log.Println("cache miss", normalizedURL)
	call.liveURL, call.err = RealGetYoutubeLiveM3U8(normalizedURL)
	if call.err != nil {
		log.Println(call.err)
		log.Println("[YTDL]", call.liveURL)
		if shouldBackoffYtdl(call.err) {
			setYtdlFailure(normalizedURL, call.err)
		} else {
			clearYtdlFailure(normalizedURL)
		}
		return "", call.err
	}
	clearYtdlFailure(normalizedURL)
	global.URLCache.Store(normalizedURL, call.liveURL)
	return call.liveURL, nil
}

func RealGetYoutubeLiveM3U8(youtubeURL string) (string, error) {
	youtubeURL = normalizeYoutubeURL(youtubeURL)
	YtdlCmd, err := GetConfig("ytdl_cmd")
	if err != nil {
		log.Println(err)
		return "", err
	}
	YtdlCmd = strings.TrimSpace(YtdlCmd)
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
	ytdlArgs := splitArgs(strings.TrimSpace(YtdlArgs))
	hasURLArg := false
	for i, v := range ytdlArgs {
		if strings.EqualFold(v, "{url}") {
			ytdlArgs[i] = youtubeURL
			hasURLArg = true
		}
	}
	if !hasURLArg {
		ytdlArgs = append(ytdlArgs, youtubeURL)
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
			wrappedErr := waitErr
			if parsedErr := buildYtdlError(waitErr, stderrBytes); parsedErr != nil {
				wrappedErr = parsedErr
			}
			log.Println(wrappedErr)
			return "", wrappedErr
		}
		if len(stderrBytes) > 0 {
			log.Println("[YTDL stderr]", string(stderrBytes))
		}
		return strings.TrimSpace(string(stdoutBytes)), nil
	}
}

func getOrCreateYtdlCall(cacheKey string) (*ytdlCall, bool) {
	ytdlCallMu.Lock()
	defer ytdlCallMu.Unlock()
	if call, ok := ytdlInflightCalls[cacheKey]; ok {
		return call, true
	}
	call := &ytdlCall{done: make(chan struct{})}
	ytdlInflightCalls[cacheKey] = call
	return call, false
}

func finishYtdlCall(cacheKey string, call *ytdlCall) {
	ytdlCallMu.Lock()
	delete(ytdlInflightCalls, cacheKey)
	ytdlCallMu.Unlock()
	close(call.done)
}

func getYtdlFailure(cacheKey string) error {
	ytdlCallMu.Lock()
	defer ytdlCallMu.Unlock()
	failure, ok := ytdlFailures[cacheKey]
	if !ok {
		return nil
	}
	if time.Now().After(failure.until) {
		delete(ytdlFailures, cacheKey)
		return nil
	}
	return failure.err
}

func setYtdlFailure(cacheKey string, err error) {
	ytdlCallMu.Lock()
	defer ytdlCallMu.Unlock()
	ytdlFailures[cacheKey] = ytdlFailure{
		err:   fmt.Errorf("yt-dlp temporary backoff for 1m: %w", err),
		until: time.Now().Add(ytdlFailureBackoff),
	}
}

func clearYtdlFailure(cacheKey string) {
	ytdlCallMu.Lock()
	defer ytdlCallMu.Unlock()
	delete(ytdlFailures, cacheKey)
}

func resetYtdlFailureState() {
	ytdlCallMu.Lock()
	defer ytdlCallMu.Unlock()
	ytdlInflightCalls = make(map[string]*ytdlCall)
	ytdlFailures = make(map[string]ytdlFailure)
}

func shouldBackoffYtdl(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "http error 429") ||
		strings.Contains(message, "too many requests") ||
		strings.Contains(message, "sign in to confirm") ||
		strings.Contains(message, "not a bot") ||
		strings.Contains(message, "visitor data")
}

func buildYtdlError(waitErr error, stderrBytes []byte) error {
	stderrText := strings.TrimSpace(string(stderrBytes))
	if stderrText == "" {
		return waitErr
	}
	lowerText := strings.ToLower(stderrText)
	if strings.Contains(lowerText, "sign in to confirm you’re not a bot") ||
		strings.Contains(lowerText, "sign in to confirm you're not a bot") ||
		strings.Contains(lowerText, "http error 429") ||
		strings.Contains(lowerText, "too many requests") {
		return fmt.Errorf("youtube requires cookies or visitor data: %w", waitErr)
	}
	return fmt.Errorf("%w: %s", waitErr, stderrText)
}

func normalizeYoutubeURL(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimFunc(s, func(r rune) bool {
		switch r {
		case '`', '\'', '"', '“', '”', '‘', '’', '<', '>':
			return true
		default:
			return unicode.IsSpace(r)
		}
	})
	return strings.TrimSpace(s)
}

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
