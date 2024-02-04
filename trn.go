package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
	"github.com/tengfei-xy/go-log"
)

type trnStruct struct {
	seller_id  string
	url        string
	trn        string
	primary_id string
	status     int
}

const MYSQL_TRN_STATUS_INSERT int = MYSQL_SELLER_STATUS_INSERT
const MYSQL_TRN_STATUS_OK int = 1
const MYSQL_TRN_STATUS_NULL int = 2
const MYSQL_TRN_STATUS_OTHER int = 3
const MYSQL_TRN_STATUS_SPECIAL int = 4

func (trn *trnStruct) start() error {
	_, err := app.db.Exec("UPDATE seller SET app = ? WHERE status = ? and (app=? or app=?) LIMIT 100", app.Basic.App_id, MYSQL_SELLER_STATUS_INSERT, 0, app.Basic.App_id)
	if err != nil {
		log.Errorf("更新seller表失败,%v", err)
		return err
	}
	return nil
}

func (trn *trnStruct) main() error {
	if !app.Exec.Enable.Trn {
		log.Info("跳过 TRN")
		return nil
	}

	app.update(MYSQL_APPLICATION_STATUS_TRN)

	log.Infof("------------------------")
	log.Infof("3. 开始 根据商家页获取TRN")
	trn.start()

	_, err := app.db.Exec("UPDATE seller SET app = ? WHERE status = ? and app=? LIMIT 100", app.Basic.App_id, MYSQL_TRN_STATUS_INSERT, 0)
	if err != nil {
		log.Errorf("更新seller表失败,%v", err)
		return err
	}
	row, err := app.db.Query("select id,seller_id from seller where status =? and app=?", MYSQL_SELLER_STATUS_INSERT, app.Basic.App_id)
	switch err {
	case nil:
		break
	case sql.ErrNoRows:
		log.Warn("没有合适的商家ID需要检查")
		return nil
	default:
		log.Error(err)
		return err

	}

	for row.Next() {
		if err := row.Scan(&trn.primary_id, &trn.seller_id); err != nil {
			log.Error(err)
			continue
		}
		trn.url = fmt.Sprintf("https://%s/sp?ie=UTF8&seller=%s", app.Domain, trn.seller_id)

		if err := robot.IsAllow(userAgent, trn.url); err != nil {
			log.Errorf("%v", err)
			continue
		}

		for err := trn.request(); err != nil; {
			log.Error(err)
			sleep(120)
		}

		trn.check()
		if err := trn.update(); err != nil {
			log.Error(err)
			continue
		}
		if trn.status == MYSQL_TRN_STATUS_OK {
			log.Infof("中国TRN: %s", trn.trn)
		}
	}
	log.Infof("3. 结束 根据商家页获取TRN")
	log.Infof("------------------------")

	return nil
}

// 作用: 根据 trn.url 查找trn
// 举例: 根据 https://www.amazon.co.uk/sp?ie=UTF8&seller=A272CUATTYX3C4
//
//	找到 91440101MA9Y624U3K
func (trn *trnStruct) request() error {

	log.Infof("查找TRN 链接: %s", trn.url)

	client := get_client()
	req, err := http.NewRequest("GET", trn.url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authority", app.Domain)
	req.Header.Set("Accept", `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7`)
	req.Header.Set("Accept-Language", `zh-CN,zh;q=0.9`)
	req.Header.Set("cache-control", `max-age=0`)
	req.Header.Set("device-memory", `8`)
	req.Header.Set("downlink", `1.5'`)
	req.Header.Set("dpr", `2`)
	req.Header.Set("ect", `3g`)
	req.Header.Set("rtt", `350`)
	if _, err := app.get_cookie(); err != nil {
		log.Error(err)
	} else {
		req.Header.Set("Cookie", app.cookie)
	}
	req.Header.Set("upgrade-insecure-requests", `1`)
	req.Header.Set("Referer", fmt.Sprintf("https://%s/?k=Hardware+electricia%%27n&crid=3CR8DCX0B3L5U&sprefix=hardware+electricia%%27n%%2Caps%%2C714&ref=nb_sb_noss", app.Domain))
	req.Header.Set("Sec-Fetch-Dest", `empty`)
	req.Header.Set("Sec-Fetch-Mode", `cors`)
	req.Header.Set("Sec-Fetch-Site", `same-origin`)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("sec-ch-ua", `"Not.A/Brand";v="8", "Chromium";v="114", "Google Chrome";v="114"`)
	req.Header.Set("sec-ch-ua-mobile", `?0`)
	req.Header.Set("sec-ch-ua-platform", `"macOS"`)

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("内部错误:%v", err)
		return err

	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		break
	case 404:
		return ERROR_NOT_404
	case 503:
		return ERROR_NOT_503
	default:
		return fmt.Errorf("状态码:%d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("内部错误:%v", err)
	}

	doc.Find("h3").Each(func(i int, s *goquery.Selection) {
		if s.Text() != "Detailed Seller Information" {
			return
		}
		s.Parent().Parent().Find("span").Each(func(i int, d *goquery.Selection) {
			text := strings.TrimSpace(d.Text())
			// log.Debugf("%d=%s", i, text)
			if text == "Trade Register Number:" {
				trn.trn = strings.TrimSpace(d.Next().Text())
				return
			}
		})
	})

	return nil
}
func (trn *trnStruct) check() {
	trn.trn = strings.ReplaceAll(trn.trn, "(1-1)", "")
	if len(trn.trn) == 0 {
		log.Errorf("检查结果:TRN为空")
		trn.status = MYSQL_TRN_STATUS_NULL
		return
	} else if len(trn.trn) < 18 {
		log.Infof("检查结果:非中国 TRN: %s", trn.trn)
		trn.status = MYSQL_TRN_STATUS_OTHER
		return
	} else if len(trn.trn) > 18 {
		log.Infof("检查结果:非中国 TRN: %s", trn.trn)
		trn.trn = ""
		trn.status = MYSQL_TRN_STATUS_OTHER
		return
	}
	if trn.trn[0] != '9' {
		log.Errorf("检查结果:18位长,非9开头 TRN: %s", trn.trn)
		trn.status = MYSQL_TRN_STATUS_SPECIAL
		return
	}
	trn.status = MYSQL_TRN_STATUS_OK
	return
}
func (trn *trnStruct) update() error {
	_, err := app.db.Exec("update seller set status=?,trn=? where id=? and app=?", trn.status, trn.trn, trn.primary_id, app.Basic.App_id)
	return err
}
