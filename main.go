package main

import (
	"io/ioutil"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/djimenez/iconv-go"
	"github.com/imroc/req"
)

var header req.Header = req.Header{
	"Host":                      "tieba.baidu.com",
	"Connection":                "keep-alive",
	"Cache-Control":             "max-age=0",
	"Upgrade-Insecure-Requests": "1",
	"User-Agent":                "Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.182 Safari/537.36 Edg/88.0.705.74",
	"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
	"Accept-Encoding":           "gzip, deflate",
	"Accept-Language":           "zh-TW,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6,zh-CN;q=0.5",
}

type signResp struct {
	Data map[interface{}]interface{} `json:"data"`
	Err  string                      `json:"error"`
	No   int                         `json:"no"`
}

func signForum(abs string, kw string, idx int, sum int) {
	param := req.Param{
		"ie":  "utf-8",
		"kw":  kw,
		"abs": abs,
	}
	r, err := req.Post("http://tieba.baidu.com/sign/add", param, header)
	if err != nil {
		log.Println(err)
	}

	// 分析尝试签到的结果
	var signresp signResp
	r.ToJSON(&signresp)

	if signresp.No == 0 {
		log.Printf("[%d/%d] %s signed success.\n", idx+1, sum, kw)
	} else {
		log.Printf("[%d/%d] %s signed fail.\n", idx+1, sum, kw)
		log.Println(signresp.Err)
	}

}

// 获取某个贴吧的abs,即签到验证信息
func getAbs(kw string) string {
	param := req.Param{
		"kw": url.QueryEscape(kw),
	}
	r, err := req.Get("http://tieba.baidu.com/f", param, header)
	if err != nil {
		log.Println(err)
	}

	html, _ := r.ToBytes()

	// 用正则来获取abs，golang不自持perl的高级re语法，所以得两次
	re0 := regexp.MustCompile(`'tbs': ".*"`)
	re1 := regexp.MustCompile(`(\d|\w){4,}`)

	temp0 := re0.Find(html)
	abs := re1.Find(temp0)

	return string(abs)
}

// 便于gb2312和utf8之间的转换
func getDom(r *req.Resp) goquery.Document {
	rs, err := r.ToString()
	if err != nil {
		log.Println(err)
	}

	rs, err = iconv.ConvertString(rs, "gb2312", "utf-8")
	if err != nil {
		log.Println(err)
	}

	dom, err := goquery.NewDocumentFromReader(strings.NewReader(rs))
	if err != nil {
		log.Println(err)
	}

	return *dom
}

// 获取贴吧列表时，获取最后一页的页码
func getPn() int {
	r, err := req.Get("http://tieba.baidu.com/f/like/mylike", header)
	if err != nil {
		log.Println(err)
	}

	dom := getDom(r)

	// 分析href里的页码
	uri, ex := dom.Find("#j_pagebar > div > a:last-child").Attr("href")
	if ex == false {
		log.Println("Not exist.")
	}

	u, err := url.Parse(uri)
	if err != nil {
		log.Println(err)
	}

	// 获取查询字符串，即末页页码
	pn, err := strconv.Atoi(u.Query().Get("pn"))
	if err != nil {
		log.Println(err)
	}

	return pn
}

// 获取贴吧列表的kw，即贴吧名
func getForums(pn int) []string {
	var forums []string

	// 一页一页地搜集贴吧名称，根据最后一页的值
	for px := 1; px <= pn; px++ {
		param := req.Param{
			"pn": px,
		}
		r, err := req.Get("http://tieba.baidu.com/f/like/mylike", param, header)
		if err != nil {
			log.Println(err)
		}

		dom := getDom(r)

		dom.Find("body > div.forum_main > div.forum_table > table > tbody > tr > td:nth-child(1) > a").
			Each(func(i int, s *goquery.Selection) {
				forums = append(forums, s.Text())
			})
	}

	return forums
}

// 装载cookie，还不能直接弄密码登录，水平不足
func loadCookie(cookie []byte) {
	if len(cookie) == 0 {
		panic("Please fill cookie.txt")
	}

	// 避免cookie末尾的换行
	cookiestr := string(cookie)
	cookiestr = strings.ReplaceAll(string(cookiestr), "\r\n", "")
	header["Cookie"] = cookiestr
}

func signAllForums(cookie []byte) {
	loadCookie(cookie)
	
	forums := getForums(getPn())
	// fmt.Println(forums)

	log.Printf("Scanned %d forums.", len(forums))
	log.Println(forums)

	// 也许可以用goroutimes来并发签到
	rest := len(forums)

	// 一个一个签到
	for i, forum := range forums {
		abs := getAbs(forum)
		log.Printf("Abs of %s is %s\n", forum, abs)
		signForum(abs, forum, i, rest)
		log.Println()
	}

	log.Println("Sign end.")
}

func main() {
	cookie, err := ioutil.ReadFile("cookie.txt")
	if err != nil {
		log.Println(err)
	}
	for {
		log.Println("Now hour is", time.Now().Hour())

		if time.Now().Hour() == 9 {
			log.Println("Start to sign.")
			signAllForums(cookie)
		} else {
			log.Println("No need sign. Sleep for 1 hour.")
		}

		time.Sleep(time.Hour)
	}
}
