package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"github.com/PuerkitoBio/goquery"
	"github.com/imroc/req"
	"github.com/djimenez/iconv-go"
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

func sign(abs string, kw string) {
	param := req.Param{
		"ie":  "utf-8",
		"kw":  kw,
		"abs": abs,
	}
	r, err := req.Post("http://tieba.baidu.com", param, header)
	if err != nil {
		log.Println(err)
	}

	var signresp signResp
	r.ToJSON(&signresp)

	if signresp.No == 0 {
		fmt.Printf("%s signed success.\n", kw)
	} else {
		fmt.Printf("%s signed fail.\n", kw)
		log.Println(signresp.Err)
	}
}

// 获取某个贴吧的abs,即签到验证信息
func getAbs(kw string) string {
	param := req.Param{
		"kw": kw,
	}
	r, err := req.Get("http://tieba.baidu.com", param, header)
	if err != nil {
		log.Println(err)
	}

	html, _ := r.ToBytes()

	re0 := regexp.MustCompile(`'tbs': ".*"`)
	re1 := regexp.MustCompile(`(\d|\w)+{4,}`)

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

func getPn() int {
	r, err := req.Get("http://tieba.baidu.com/f/like/mylike", header)
	if err != nil {
		log.Println(err)
	}

	dom := getDom(r)

	uri, ex := dom.Find("#j_pagebar > div > a:last-child").Attr("href")
	if ex == false {
		fmt.Println("Not exist.")
	}

	u, err := url.Parse(uri)
	if err != nil {
		log.Println(err)
	}

	pn, err := strconv.Atoi(u.Query().Get("pn"))
	if err != nil {
		log.Println(err)
	}

	return pn
}

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

func loadCookie() {
	cookie, err := ioutil.ReadFile("cookie.txt")
	if err != nil {
		log.Println(err)
	}

	// 避免cookie末尾的换行
  cookiestr := string(cookie)
  cookiestr = strings.ReplaceAll(string(cookiestr), "\r\n", "")
	header["Cookie"] = cookiestr
}

func main() {
	loadCookie()
	forums := getForums(getPn())
	fmt.Println(forums)
}
