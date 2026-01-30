package service

import (
	"context"
	"io"
	"log"
	"os/exec"
	"strings"

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
	ytdlArgs := strings.Fields(YtdlArgs)
	for i, v := range ytdlArgs {
		if strings.EqualFold(v, "{url}") {
			ytdlArgs[i] = youtubeURL
		}
	}
	_, err = exec.LookPath(YtdlCmd)
	if err != nil {
		log.Println(err)
		return "", err
	} else {
		ctx, cancelFunc := context.WithTimeout(context.Background(), global.HttpClientTimeout)
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
		stdoutBytes, err := io.ReadAll(stdout)
		if err != nil {
			log.Println(err)
			return "", err
		}
		stderrBytes, err := io.ReadAll(stderr)
		if err != nil {
			log.Println(err)
			return "", err
		}
		if err := cmd.Wait(); err != nil {
			log.Println("[YTDL stderr]", string(stderrBytes))
			log.Println(err)
			return "", err
		}
		if len(stderrBytes) > 0 {
			log.Println("[YTDL stderr]", string(stderrBytes))
		}
		return strings.TrimSpace(string(stdoutBytes)), nil
	}
}
