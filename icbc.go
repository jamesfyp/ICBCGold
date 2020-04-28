package main

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"io/ioutil"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// URLGold 工商的黄金价格URL
var URLGold = "http://www.icbc.com.cn/ICBCDynamicSite/Charts/GoldTendencyPicture.aspx"

type BarkToken struct {
	Token []string `mapstructure:"token"`
}

var barkConf = &BarkToken{}

// Cache 缓存, 设置 告警的阈值 , +-0.5
type Cache struct {
	Alarm float64
}

var cache = &Cache{366}

// LogSetup 初始化日志
func LogSetup() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.With().Caller().Logger().Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	})
}

// IcbcGold 查询黄金价格
func IcbcGold() {
	var (
		res        *http.Response
		err        error
		doc        *goquery.Document
		httpClient http.Client
		jar        *cookiejar.Jar
		price      float64
	)

	// 处理cookies, 这里用不到保持session
	jar, _ = cookiejar.New(nil)
	httpClient.Jar = jar

	if res, err = httpClient.Get(URLGold); err != nil {
		log.Error().Msgf("请求失败, %v", err.Error())
		return
	}
	defer res.Body.Close()
	if doc, err = goquery.NewDocumentFromReader(res.Body); err != nil {
		log.Error().Msgf("goquery解析失败, %v", err.Error())
		return
	}
	// Attr 获取属性
	flag := false
	doc.Find(`#TABLE1 > tbody > tr:nth-child(2) > td:nth-child(3)`).Each(func(i int, s *goquery.Selection) {
		flag = true
		price, _ = strconv.ParseFloat(strings.TrimSpace(s.Text()), 64)
		log.Info().Msgf("当前价格: %v, 告警阈值: %v", price, cache.Alarm)
		Alarm(price)
	})
	if !flag {
		log.Error().Msgf("没有获取到黄金价格")
	}
	// fmt.Println(doc.Find("#TABLE1"))
}

// Alarm 判断价格
func Alarm(price float64) {

	inc := int(math.Abs(price-cache.Alarm) / 0.5)
	if inc >= 1 {
		if price-cache.Alarm > 0 {
			if cache.Alarm != 0 {
				go wechat(fmt.Sprintf("当前价格: %v [上升]", price))
				go sendBark(fmt.Sprintf("当前价格: %v [上升]", price))
				log.Info().Msgf("当前价格: %v [上升]", price)
			}
			cache.Alarm = cache.Alarm + float64(inc)*0.5

		} else {
			cache.Alarm = cache.Alarm - float64(inc)*0.5
			log.Info().Msgf("当前价格: %v [下降]", price)
			go wechat(fmt.Sprintf("当前价格: %v [下降]", price))
			go sendBark(fmt.Sprintf("当前价格: %v [下降]", price))
		}
	}

}

func wechat(msg string) {
	var (
		res *http.Response
		err error
	)
	if res, err = http.PostForm("http://api.xxxx.com/weixin", url.Values{"msg": {msg}}); err != nil {
		log.Error().Msg("wechat: 发送失败")
	} else {
		log.Info().Msgf("wechat: 发送成功")
	}
	res.Body.Close()
}

func sendBark(msg string) {
	var (
		res   *http.Response
		err   error
		token string
	)

	for _, token = range barkConf.Token {
		if res, err = http.Get(fmt.Sprintf("https://api.day.app/%s/黄金价格/%s", token, msg)); err != nil {
			log.Error().Msgf("token:%s, 发送失败", token)

		} else {
			log.Info().Msgf("token:%s, 发送成功", token)
		}
		res.Body.Close()
	}

}

func run() {
	for {
		go IcbcGold()
		time.Sleep(time.Minute * 1)
	}
}

//Setup 初始配置
func Setup() {
	LogSetup()
	viper.SetConfigType("YAML")
	// 读取配置文件内容
	data, err := ioutil.ReadFile("icbc.yaml")
	if err != nil {
		log.Error().Msgf("Read 'icbc.yaml' fail: %v\n", err)
	}
	viper.ReadConfig(bytes.NewBuffer(data))
	viper.UnmarshalKey("bark", barkConf)
	log.Info().Msg("初始化配置")
	for _, token := range barkConf.Token {
		log.Info().Msgf("barkToken: %s", token)
	}

}

func main() {
	Setup()
	go run()
	// go http.ListenAndServe("0.0.0.0:6060", nil)
	select {}
}
