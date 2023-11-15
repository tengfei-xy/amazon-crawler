package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	log "github.com/tengfei-xy/go-log"
)

const MYSQL_SEARCH_STATUS_START int64 = 0
const MYSQL_SEARCH_STATUS_OVER int64 = 1

type search struct {
	zh_key        string
	en_key        string
	category_id   int64
	url           string
	start         int
	end           int
	html          string
	valid         int
	product_url   string
	product_param string
}

func (s *search) main() error {
	app.update(MYSQL_APPLICATION_STATUS_SEARCH)

	log.Infof("------------------------")
	log.Infof("1. 开始搜索关键词")
	row, err := app.db.Query(`select id,zh_key,en_key from category order by priority DESC `)
	if err != nil {
		return err
	}
	s.start = 1
	s.end = 10
	for row.Next() {
		row.Scan(&s.category_id, &s.zh_key, &s.en_key)
		s.en_key = s.set_en_key()
		insert_id, err := s.search_start()
		if err != nil {
			log.Errorf("插入失败 关键词:%s %v", s.zh_key, err)
			continue
		}
		for ; s.start < s.end; s.start++ {
			h, err := s.NewRequest(s.start)
			if err != nil {
				log.Error(err)
				continue
			}
			s.get_product_url(h)
		}
		err = s.search_end(insert_id)
		if err != nil {
			log.Errorf("更新结果失败 关键词:%s %v", s.zh_key, err)
			continue
		}
		s.start = 1
	}
	log.Infof("------------------------")
	return nil
}
func (s *search) search_start() (int64, error) {
	r, err := app.db.Exec("insert into search_statistics(category_id,app) values(?,?)", s.category_id, app.Identified.App)
	if err != nil {
		return 0, err
	}

	id, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}
	log.Infof("开始搜索 关键词:%s 关键词ID:%d 状态:%d(开始)", s.zh_key, s.category_id, MYSQL_SEARCH_STATUS_START)
	return id, nil
}
func (s *search) search_end(insert_id int64) error {
	_, err := app.db.Exec("update search_statistics set status=?,end=CURRENT_TIMESTAMP,valid=? where id=?", MYSQL_SEARCH_STATUS_OVER, s.valid, insert_id)
	if err != nil {
		return err
	}
	log.Infof("搜索完成 关键词:%s 完成ID:%d 有效数:%d", s.zh_key, insert_id, s.valid)
	return nil
}
func (s *search) set_en_key() string {
	return strings.ReplaceAll(strings.ReplaceAll(s.en_key, " ", "+"), "'", "%27")
}
func (s *search) NewRequest(seq int) (*goquery.Document, error) {
	// 	curl 'https://www.amazon.co.uk/Povxlum-Waterproof-Shockproof-Electrician-27X22X7Cm/dp/B0C7VF6B53/ref=sr_1_1?crid=2V9436DZJ6IJF&keywords=Hardware+electrician&qid=1699939808&sprefix=clothe%2Caps%2C552&sr=8-1' \
	//   -H 'authority: www.amazon.co.uk' \
	//   -H 'accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7' \
	//   -H 'accept-language: zh-CN,zh;q=0.9' \
	//   -H 'cache-control: no-cache' \
	//   -H 'device-memory: 8' \
	//   -H 'downlink: 1.55' \
	//   -H 'dpr: 2' \
	//   -H 'ect: 3g' \
	//   -H 'pragma: no-cache' \
	//   -H 'rtt: 400' \
	//   -H 'sec-ch-device-memory: 8' \
	//   -H 'sec-ch-dpr: 2' \
	//   -H 'sec-ch-ua: "Google Chrome";v="119", "Chromium";v="119", "Not?A_Brand";v="24"' \
	//   -H 'sec-ch-ua-mobile: ?0' \
	//   -H 'sec-ch-ua-platform: "macOS"' \
	//   -H 'sec-ch-ua-platform-version: "14.1.0"' \
	//   -H 'sec-ch-viewport-width: 2028' \
	//   -H 'sec-fetch-dest: document' \
	//   -H 'sec-fetch-mode: navigate' \
	//   -H 'sec-fetch-site: same-origin' \
	//   -H 'sec-fetch-user: ?1' \
	//   -H 'upgrade-insecure-requests: 1' \
	//   -H 'user-agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36' \
	//   -H 'viewport-width: 2028' \
	//   --compressed

	url := fmt.Sprintf("https://www.amazon.co.uk/s?k=%s&page=%d&crid=2V9436DZJ6IJF&qid=1699839233&sprefix=clothe%%2Caps%%2C552&ref=sr_pg_2", s.en_key, seq)
	log.Infof("开始搜索 关键词:%s 页面:%d url:%s", s.zh_key, seq, url)

	client := get_client()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authority", `www.amazon.co.uk`)
	req.Header.Set("Accept", `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7`)
	req.Header.Set("Accept-Language", `zh-CN,zh;q=0.9`)
	req.Header.Set("cache-control", `max-age=0`)
	req.Header.Set("device-memory", `8`)
	req.Header.Set("device-memory", `8`)
	req.Header.Set("downlink", `1.55'`)
	req.Header.Set("dpr", `2`)
	req.Header.Set("ect", `3g`)
	req.Header.Set("pragma", `400`)
	// req.Header.Set("Cookie", cookie)
	req.Header.Set("upgrade-insecure-requests", `1`)
	req.Header.Set("Referer", "https://www.amazon.co.uk/s?k=Hardware+electricia%27n&crid=3CR8DCX0B3L5U&sprefix=hardware+electricia%27n%2Caps%2C714&ref=nb_sb_noss")
	req.Header.Set("Sec-Fetch-Dest", `empty`)
	req.Header.Set("Sec-Fetch-Mode", `cors`)
	req.Header.Set("Sec-Fetch-Site", `same-origin`)
	req.Header.Set("User-Agent", `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36`)
	req.Header.Set("sec-ch-ua", `"Not.A/Brand";v="8", "Chromium";v="114", "Google Chrome";v="114"`)
	req.Header.Set("sec-ch-ua-mobile", `?0`)
	req.Header.Set("sec-ch-ua-platform", `"macOS"`)

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("内部错误:%v", err)
		return nil, err

	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Errorf("状态码:%d", resp.StatusCode)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("内部错误:%v", err)
	}
	return doc, nil
}

func (s *search) get_product_url(doc *goquery.Document) {

	defer func() {
		if err := recover(); err != nil {
			return
		}
	}()

	res := doc.Find("div[class~=s-search-results]").First()

	res.Find("div[data-index]").Each(func(i int, h *goquery.Selection) {
		// 处理找到的 div 元素
		link, exist := h.Find("a").First().Attr("href")
		if !exist {
			return
		}
		if strings.HasPrefix(link, "/s") || strings.HasPrefix(link, "/gp/") {
			return
		}
		url := strings.Split(link, "/ref=")
		_, err := app.db.Exec(`INSERT INTO product(url,param) values(?,?)`, url[0], "/ref="+url[1])

		if is_duplicate_entry(err) {
			log.Infof("商品已存在 关键词:%s 链接:%s ", s.zh_key, link)
			return
		}
		if err != nil {
			log.Errorf("商品插入失败 关键词:%s 链接:%s %v ", s.zh_key, link, err)
			return
		}

		log.Infof("商品插入成功 关键词:%s 链接:%s ", s.zh_key, link)
		s.valid += 1

		return
	})
	return
}
