package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	log "github.com/tengfei-xy/go-log"
)

const MYSQL_SEARCH_STATUS_START int64 = 0
const MYSQL_SEARCH_STATUS_OVER int64 = 1

type search struct {
	zh_key      string
	en_key      string
	category_id int64
	url         string
	start       int
	end         int
	html        string
	valid       int
}

func (s *search) main() error {
	app.update(MYSQL_APPLICATION_STATUS_SEARCH)

	log.Infof("------------------------")
	log.Infof("1. 开始搜索关键词")
	row, err := app.db.Query(`select id,zh_key,en_key from category order by priority`)
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
func (s *search) NewRequest(seq int) (string, error) {
	url := fmt.Sprintf("https://www.amazon.co.uk/s?k=%s&page=%d&crid=2V9436DZJ6IJF&qid=1699839233&sprefix=clothe%%2Caps%%2C552&ref=sr_pg_2", s.en_key, seq)
	log.Infof("开始搜索 关键词:%s 页面:%d url:%s", s.zh_key, seq, url)

	// 创建一个新的上下文
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// 运行任务
	var htmlContent string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.OuterHTML("html", &htmlContent),
	)
	if err != nil {
		return "", err
	}
	s.html = htmlContent
	// 打印最终的 HTML 代码
	return htmlContent, nil
}

func (s *search) get_product_url(body string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		log.Errorf("内部错误:%v", err)
		return
	}
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(body)
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
			log.Infof("已存在 关键词:%s 链接:%s ", s.zh_key, link)
			return
		}
		if err != nil {
			log.Errorf("插入失败 关键词:%s 链接:%s %v ", s.zh_key, link, err)
			return
		}

		log.Infof("插入成功 关键词:%s 链接:%s ", s.zh_key, link)
		s.valid += 1
	})
}
