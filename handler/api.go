package handler

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/qist/livetv/model"
	"github.com/qist/livetv/service"
	"github.com/qist/livetv/util"

	"golang.org/x/text/language"
)

var langMatcher = language.NewMatcher([]language.Tag{
	language.English,
	language.Chinese,
})

func IndexHandler(c *gin.Context) {
	if sessions.Default(c).Get("logined") != true {
		c.Redirect(http.StatusFound, "/login")
	}
	acceptLang := c.Request.Header.Get("Accept-Language")
	langTag, _ := language.MatchStrings(langMatcher, acceptLang)

	baseUrl, err := service.GetConfig("base_url")
	if err != nil {
		log.Println(err.Error())
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"ErrMsg": err.Error(),
		})
		return
	}
	m3uFilename, err := service.GetConfig("m3u_filename")
	if err != nil {
		m3uFilename = "lives.m3u"
	}
	channelParam, err := service.GetConfig("channel_param")
	if err != nil {
		channelParam = "c"
	}
	channelModels, err := service.GetAllChannel()
	if err != nil {
		log.Println(err.Error())
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"ErrMsg": err.Error(),
		})
		return
	}
	var m3uName string
	// Check if the matched language is any variant of Chinese
	isChinese := false
	langStr := langTag.String()
	if strings.HasPrefix(langStr, "zh") || langStr == "zh" {
		isChinese = true
	}
	if isChinese {
		m3uName = "M3U 频道列表"
	} else {
		m3uName = "M3U File"
	}
	channels := make([]Channel, len(channelModels)+1)
	channels[0] = Channel{
		ID:   0,
		Name: m3uName,
		M3U8: baseUrl + "/" + m3uFilename,
	}
	for i, v := range channelModels {
		channelID := strconv.Itoa(int(v.ID))
		if v.CustomID != "" {
			channelID = v.CustomID
		}
		channels[i+1] = Channel{
			ID:       v.ID,
			CustomID: v.CustomID,
			Name:     v.Name,
			URL:      v.URL,
			M3U8:     baseUrl + "/live.m3u8?" + channelParam + "=" + channelID,
			Proxy:    v.Proxy,
		}
	}
	conf, err := loadConfig()
	if err != nil {
		log.Println(err.Error())
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"ErrMsg": err.Error(),
		})
		return
	}

	var templateFilename string
	if isChinese {
		templateFilename = "index-zh.html"
	} else {
		templateFilename = "index.html"
	}
	c.HTML(http.StatusOK, templateFilename, gin.H{
		"Channels": channels,
		"Configs":  conf,
	})
}

func loadConfig() (Config, error) {
	var conf Config
	if cmd, err := service.GetConfig("ytdl_cmd"); err != nil {
		return conf, err
	} else {
		conf.Cmd = cmd
	}
	if args, err := service.GetConfig("ytdl_args"); err != nil {
		return conf, err
	} else {
		conf.Args = args
	}
	if burl, err := service.GetConfig("base_url"); err != nil {
		return conf, err
	} else {
		conf.BaseURL = burl
	}
	if m3uFilename, err := service.GetConfig("m3u_filename"); err != nil {
		conf.M3UFilename = "lives.m3u"
	} else {
		conf.M3UFilename = m3uFilename
	}
	if channelParam, err := service.GetConfig("channel_param"); err != nil {
		conf.ChannelParam = "c"
	} else {
		conf.ChannelParam = channelParam
	}
	return conf, nil
}

func NewChannelHandler(c *gin.Context) {
	if sessions.Default(c).Get("logined") != true {
		c.Redirect(http.StatusFound, "/login")
	}
	chName := c.PostForm("name")
	chURL := c.PostForm("url")
	chCustomID := c.PostForm("custom_id")
	if chName == "" || chURL == "" {
		c.Redirect(http.StatusFound, "/")
		return
	}
	chProxy := c.PostForm("proxy") != ""
	mch := model.Channel{
		CustomID: chCustomID,
		Name:     chName,
		URL:      chURL,
		Proxy:    chProxy,
	}
	err := service.SaveChannel(mch)
	if err != nil {
		log.Println(err.Error())
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"ErrMsg": err.Error(),
		})
		return
	}
	c.Redirect(http.StatusFound, "/")
}

func DeleteChannelHandler(c *gin.Context) {
	if sessions.Default(c).Get("logined") != true {
		c.Redirect(http.StatusFound, "/login")
	}
	chID := util.String2Uint(c.Query("id"))
	if chID == 0 {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"ErrMsg": "empty id",
		})
		return
	}
	err := service.DeleteChannel(chID)
	if err != nil {
		log.Println(err.Error())
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"ErrMsg": err.Error(),
		})
		return
	}
	c.Redirect(http.StatusFound, "/")
}

func UpdateConfigHandler(c *gin.Context) {
	if sessions.Default(c).Get("logined") != true {
		c.Redirect(http.StatusFound, "/login")
	}
	ytdlCmd := c.PostForm("cmd")
	ytdlArgs := c.PostForm("args")
	baseUrl := strings.TrimSuffix(c.PostForm("baseurl"), "/")
	m3uFilename := c.PostForm("m3u_filename")
	channelParam := c.PostForm("channel_param")
	if len(ytdlCmd) > 0 {
		err := service.SetConfig("ytdl_cmd", ytdlCmd)
		if err != nil {
			log.Println(err.Error())
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"ErrMsg": err.Error(),
			})
			return
		}
	}
	if len(ytdlArgs) > 0 {
		err := service.SetConfig("ytdl_args", ytdlArgs)
		if err != nil {
			log.Println(err.Error())
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"ErrMsg": err.Error(),
			})
			return
		}
	}
	if len(baseUrl) > 0 {
		err := service.SetConfig("base_url", baseUrl)
		if err != nil {
			log.Println(err.Error())
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"ErrMsg": err.Error(),
			})
			return
		}
	}
	if len(m3uFilename) > 0 {
		err := service.SetConfig("m3u_filename", m3uFilename)
		if err != nil {
			log.Println(err.Error())
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"ErrMsg": err.Error(),
			})
			return
		}
	}
	if len(channelParam) > 0 {
		err := service.SetConfig("channel_param", channelParam)
		if err != nil {
			log.Println(err.Error())
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"ErrMsg": err.Error(),
			})
			return
		}
	}
	c.Redirect(http.StatusFound, "/")
}

func LogHandler(c *gin.Context) {
	if sessions.Default(c).Get("logined") != true {
		c.Redirect(http.StatusFound, "/login")
	}
	c.File(os.Getenv("LIVETV_DATADIR") + "/livetv.log")
}

func LoginViewHandler(c *gin.Context) {
	session := sessions.Default(c)
	crsfToken := util.RandString(10)
	session.Set("crsfToken", crsfToken)
	err := session.Save()
	if err != nil {
		log.Println(err.Error())
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"ErrMsg": err.Error(),
		})
		return
	}
	c.HTML(http.StatusOK, "login.html", gin.H{
		"Crsf": crsfToken,
	})
}

func LoginActionHandler(c *gin.Context) {
	session := sessions.Default(c)
	crsfToken := c.PostForm("crsf")
	if crsfToken != session.Get("crsfToken") {
		c.HTML(http.StatusOK, "error.html", gin.H{
			"ErrMsg": "Password error!",
		})
		return
	}
	pass := c.PostForm("password")
	cfgPass, err := service.GetConfig("password")
	if err != nil {
		log.Println(err.Error())
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"ErrMsg": err.Error(),
		})
		return
	}
	if pass == cfgPass {
		session.Set("logined", true)
		err = session.Save()
		if err != nil {
			log.Println(err.Error())
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"ErrMsg": err.Error(),
			})
			return
		}
		c.Redirect(http.StatusFound, "/")
	} else {
		c.HTML(http.StatusOK, "error.html", gin.H{
			"ErrMsg": "Password error!",
		})
	}
}

func LogoutHandler(c *gin.Context) {
	if sessions.Default(c).Get("logined") != true {
		c.Redirect(http.StatusFound, "/login")
	}
	session := sessions.Default(c)
	session.Delete("logined")
	err := session.Save()
	if err != nil {
		log.Println(err.Error())
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"ErrMsg": err.Error(),
		})
		return
	}
	c.Redirect(http.StatusFound, "/login")
}

func ChangePasswordHandler(c *gin.Context) {
	if sessions.Default(c).Get("logined") != true {
		c.Redirect(http.StatusFound, "/login")
	}
	pass := c.PostForm("password")
	pass2 := c.PostForm("password2")
	if pass != pass2 {
		c.HTML(http.StatusOK, "error.html", gin.H{
			"ErrMsg": "Password mismatch!",
		})
	}
	err := service.SetConfig("password", pass)
	if err != nil {
		log.Println(err.Error())
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"ErrMsg": err.Error(),
		})
		return
	}
	LogoutHandler(c)
}
