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

type sellerStruct struct {
	seller_id    string
	url          string
	businessName string
	trn          string
	address      string
	primary_id   string
	trn_status   int
	all_status   int
}

const MYSQL_SELLER_STATUS_TRN_OK int = 1
const MYSQL_SELLER_STATUS_TRN_NO int = 2
const MYSQL_SELLER_STATUS_TRN_OTHER int = 3
const MYSQL_SELLER_STATUS_TRN_SPECIAL int = 4

const MYSQL_SELLER_STATUS_INFO_INSERT int = MYSQL_PRODUCT_STATUS_INSERT
const MYSQL_SELLER_STATUS_INFO_OK int = 1
const MYSQL_SELLER_STATUS_INFO_ALL_NO_NAME int = 2
const MYSQL_SELLER_STATUS_INFO_ALL_NO_ADDRESS int = 3
const MYSQL_SELLER_STATUS_INFO_ALL_NO_TRN int = 3

func (seller *sellerStruct) prepare() {
	app.update(MYSQL_APPLICATION_STATUS_SELLER)
	log.Infof("------------------------")
	log.Infof("3. 开始 根据商家页获取商家信息")
}
func (seller *sellerStruct) over() {
	log.Infof("3. 结束 根据商家页获取商家信息")
	log.Infof("------------------------")
}
func (seller *sellerStruct) start() error {
	_, err := app.db.Exec("UPDATE seller SET app_id = ? WHERE all_status = ? and (app_id=? or app_id=?) LIMIT 100", app.Basic.App_id, MYSQL_SELLER_STATUS_INFO_INSERT, 0, app.Basic.App_id)
	if err != nil {
		log.Errorf("更新seller表失败,%v", err)
		return err
	}
	return nil
}
func (seller *sellerStruct) main() error {

	if !app.Exec.Enable.Seller {
		log.Warn("跳过 获取商家信息")
		return nil
	}
	if app.Exec.Loop.Seller == app.Exec.Loop.seller_time {
		log.Warn("已经达到执行次数 获取商家信息")
		return nil
	}

	seller.prepare()

	if app.Exec.Loop.Seller == 0 {
		log.Info("循环次数无限")
	} else {
		log.Infof("循环次数剩余:%d", app.Exec.Loop.Seller-app.Exec.Loop.seller_time)
	}
	app.Exec.Loop.seller_time++

	if err := seller.start(); err != nil {
		return err
	}

	row, err := app.db.Query("select id,seller_id from seller where all_status =? and app_id=?", MYSQL_SELLER_STATUS_INFO_INSERT, app.Basic.App_id)
	switch err {
	case nil:
		break
	case sql.ErrNoRows:
		log.Warnf("指定的app_id:%d,没有需要处理的商家信息", app.Basic.App_id)
		return nil
	default:
		log.Error(err)
		return err

	}
	for row.Next() {
		if err := row.Scan(&seller.primary_id, &seller.seller_id); err != nil {
			log.Error(err)
			continue
		}
		seller.url = fmt.Sprintf("https://%s/sp?ie=UTF8&seller=%s", app.Domain, seller.seller_id)

		if err := robot.IsAllow(userAgent, seller.url); err != nil {
			log.Errorf("%v", err)
			continue
		}

		for err := seller.request(); err != nil; {
			log.Error(err)
			sleep(120)
		}

		seller.trnCheck()
		seller.addressCheck()
		seller.nameCheck()
		if err := seller.update(); err != nil {
			log.Error(err)
			continue
		}

	}

	seller.over()
	return nil
}

// 作用: 根据 seller.url 请求商家信息
// 举例: 根据 https://www.amazon.co.uk/sp?ie=UTF8&seller=A272CUATTYX3C4 请求商家信息
func (seller *sellerStruct) request() error {

	log.Infof("请求链接 %s", seller.url)

	client := get_client()
	req, err := http.NewRequest("GET", seller.url, nil)
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

	sellerTxt := doc.Find("div#page-section-detail-seller-info").Find("span").Text()

	seller.all_status = MYSQL_SELLER_STATUS_INFO_OK
	var info []string

	for _, line := range strings.Split(sellerTxt, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "Business Name:" {
			info = append(info, line)
		} else if strings.Contains(line, "Business Name:") {
			line = strings.ReplaceAll(line, "Business Name:", "")
			info = append(info, line)
			info = append(info, "Business Type:")
		} else if strings.Contains(line, "Business Type:") {
			line = strings.ReplaceAll(line, "Business Type:", "")
			info = append(info, line)
			info = append(info, "Business Type:")
		} else if strings.Contains(line, "Trade Register Number:") {
			line = strings.ReplaceAll(line, "Trade Register Number:", "")
			info = append(info, line)
			info = append(info, "Trade Register Number:")
		} else if strings.Contains(line, "Business Address:") {
			line = strings.ReplaceAll(line, "Business Address:", "")
			info = append(info, line)
			info = append(info, "Business Address:")
		} else if strings.Contains(line, "VAT Number:") {
			line = strings.ReplaceAll(line, "VAT Number:", "")
			info = append(info, line)
			info = append(info, "VAT Number:")
		} else {
			info = append(info, line)
		}
	}

	for i, line := range info {
		if strings.Contains(line, "Business Name") {
			seller.businessName = info[i+1]
		} else if strings.Contains(line, "Trade Register Number") {
			seller.trn = info[i+1]
		} else if strings.Contains(line, "Business Address") {
			seller.address = strings.Join(info[i+1:], " ")
		} else if strings.Contains(line, "VAT Number") {
			// seller.address = strings.Join(info[i+1:], " ")
		} else if strings.Contains(line, "Business Type") {
			// seller.address = strings.Join(info[i+1:], " ")
		}
	}

	return nil
}

// 作用: 检查TRN
func (seller *sellerStruct) trnCheck() {
	seller.trn = strings.ReplaceAll(seller.trn, "(1-1)", "")
	if len(seller.trn) == 0 {
		log.Warnf("检查结果 TRN为空")
		seller.trn_status = MYSQL_SELLER_STATUS_TRN_NO
		seller.all_status = MYSQL_SELLER_STATUS_INFO_ALL_NO_TRN
		return
	} else if len(seller.trn) < 18 {
		log.Warnf("检查结果 TRN: %s (非中国)", seller.trn)
		seller.trn_status = MYSQL_SELLER_STATUS_TRN_OTHER
		return
	} else if len(seller.trn) > 18 {
		log.Warnf("检查结果 TRN: %s (非中国)", seller.trn)
		seller.trn = ""
		seller.trn_status = MYSQL_SELLER_STATUS_TRN_OTHER
		return
	}
	if seller.trn[0] != '9' {
		log.Warnf("检查结果 TRN: %s (18位长,非9开头)", seller.trn)
		seller.trn_status = MYSQL_SELLER_STATUS_TRN_SPECIAL
		return
	}
	log.Infof("查找结果 TRN: %s(中国)", seller.trn)
	seller.trn_status = MYSQL_SELLER_STATUS_TRN_OK
	return
}

func (seller *sellerStruct) addressCheck() {
	if len(seller.address) == 0 {
		log.Errorf("检查结果 地址为空")
		seller.all_status = MYSQL_SELLER_STATUS_INFO_ALL_NO_ADDRESS
		return
	}
	log.Infof("查找结果 地址: %s", seller.address)
}
func (seller *sellerStruct) nameCheck() {
	if len(seller.businessName) == 0 {
		log.Errorf("检查结果 商家名称为空")
		seller.all_status = MYSQL_SELLER_STATUS_INFO_ALL_NO_NAME
		return
	}
	log.Infof("查找结果 商家名称: %s", seller.businessName)
}
func (seller *sellerStruct) update() error {
	_, err := app.db.Exec("update seller set trn_status=?,trn=?,name=?,address=?,all_status=? where id=? and app_id=?", seller.trn_status, seller.trn, seller.businessName, seller.address, seller.all_status, seller.primary_id, app.Basic.App_id)
	return err
}
