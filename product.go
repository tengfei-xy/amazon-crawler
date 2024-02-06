package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	log "github.com/tengfei-xy/go-log"
)

type productStruct struct {
	// 产品页的商家页面链接
	url string

	// 产品页的商家ID
	id string
}

const MYSQL_PRODUCT_STATUS_INSERT int = 0
const MYSQL_PRODUCT_STATUS_CHEKCK int = 1
const MYSQL_PRODUCT_STATUS_OVER int = 2
const MYSQL_PRODUCT_STATUS_ERROR_OVER int = 3
const MYSQL_PRODUCT_STATUS_NO_PRODUCT int = 4

func (product *productStruct) main() error {
	if !app.Exec.Enable.Product {
		log.Info("跳过 产品")
		return nil
	}

	app.update(MYSQL_APPLICATION_STATUS_PRODUCT)

	log.Infof("------------------------")
	log.Infof("2. 开始从产品页获取商家ID")
	_, err := app.db.Exec("UPDATE product SET status = ? ,app = ? WHERE (status = ? or status=?) and (app=? or app=?)  LIMIT 100", MYSQL_PRODUCT_STATUS_CHEKCK, app.Basic.App_id, MYSQL_PRODUCT_STATUS_INSERT, MYSQL_PRODUCT_STATUS_ERROR_OVER, 0, app.Basic.App_id)
	if err != nil {
		log.Errorf("更新product表失败,%v", err)
		return err
	}

	row, err := app.db.Query(`select id,url,param from product where status=? and app = ?`, MYSQL_PRODUCT_STATUS_CHEKCK, app.Basic.App_id)
	if err != nil {
		log.Errorf("查询product表失败,%v", err)
		return err
	}
	for row.Next() {
		product.id = ""
		var primary_id int64
		var url, param string
		if err := row.Scan(&primary_id, &url, &param); err != nil {
			log.Errorf("获取product表的值失败,%v", err)
			continue
		}

		url = "https://" + app.Domain + url + param
		if err := robot.IsAllow(userAgent, url); err != nil {
			log.Errorf("%v", err)
			continue
		}

		log.Infof("查找商品链接 ID:%d url:%s", primary_id, url)
		err := product.request(url)
		if err != nil {
			if err == ERROR_NOT_SELLER_URL {
				product.update_status(primary_id, MYSQL_PRODUCT_STATUS_NO_PRODUCT)
				continue
			} else if err == ERROR_NOT_404 || err == ERROR_NOT_503 || err == ERROR_VERIFICATION {
				product.update_status(primary_id, MYSQL_PRODUCT_STATUS_ERROR_OVER)
				log.Error(err)
				sleep(300)
				continue
			} else {
				product.update_status(primary_id, MYSQL_PRODUCT_STATUS_ERROR_OVER)
				log.Error(err)
				sleep(300)
				continue

			}
		}

		product.get_seller_id()

		err = product.insert_selll_id()
		if is_duplicate_entry(err) {
			log.Infof("店铺已存在 商家ID:%s", product.id)
			err = nil
		}
		if err != nil {
			log.Error(err)
			continue
		}
		if err := product.update_status(primary_id, MYSQL_PRODUCT_STATUS_OVER); err != nil {
			log.Error(err)
			continue
		}

	}
	log.Infof("2. 结束从产品页获取商家ID")
	log.Infof("------------------------")

	return nil
}

func (product *productStruct) request(url string) error {
	client := get_client()
	req, err := http.NewRequest("GET", url, nil)
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

	if resp.StatusCode != 200 {
		log.Errorf("状态码:%d", resp.StatusCode)
		return err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("内部错误:%v", err)
	}

	if doc.Find("h4").First().Text() == "Enter the characters you see below" {
		return ERROR_VERIFICATION
	}

	res := doc.Find("a[id=sellerProfileTriggerId]").First()

	url, exist := res.Attr("href")
	if !exist {
		return ERROR_NOT_SELLER_URL
	}

	product.url = url

	return nil
}
func (product *productStruct) get_seller_id() string {
	for _, j := range strings.Split(product.url, "&") {
		if strings.HasPrefix(j, "seller=") {
			product.id = strings.Split(j, "seller=")[1]
		}
	}
	// if ( product.id=="")
	return product.id
}
func (product *productStruct) insert_selll_id() error {
	_, err := app.db.Exec("insert into seller (seller_id,app_id) values(?,?)", product.id, 0)
	return err
}

func (product *productStruct) update_status(id int64, s int) error {
	_, err := app.db.Exec("UPDATE product SET status = ? ,app = ? WHERE id = ?", s, app.Basic.App_id, id)
	if err != nil {
		log.Infof("更新product表状态失败 ID:%d app:%d 状态:%d", id, app.Basic.App_id, s)
		return err
	}
	log.Infof("更新product表状态成功 ID:%d 状态:%d app:%d ", id, s, app.Basic.App_id)
	return nil
}
