package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/tengfei-xy/go-log"
	"gopkg.in/yaml.v3"
)

const MYSQL_APPLICATION_STATUS_START int = 0
const MYSQL_APPLICATION_STATUS_OVER int = 1
const MYSQL_APPLICATION_STATUS_SEARCH int = 2
const MYSQL_APPLICATION_STATUS_SELLER int = 3
const MYSQL_APPLICATION_STATUS_TRN int = 4

type appConfig struct {
	Mysql      `yaml:"mysql"`
	Basic      `yaml:"basic"`
	Proxy      `yaml:"proxy"`
	Exec       `yaml:"exec"`
	db         *sql.DB
	cookie     string
	primary_id int64
}
type Exec struct {
	Enable          `yaml:"enable"`
	Search_priority int `yaml:"search_priority"`
}
type Enable struct {
	Search bool `yaml:"search"`
	Seller bool `yaml:"seller"`
	Trn    bool `yaml:"trn"`
}
type Basic struct {
	App_id  int    `yaml:"app_id"`
	Host_id int    `yaml:"host_id"`
	Test    bool   `yaml:"test"`
	Domain  string `yaml:"domain"`
}
type Proxy struct {
	Enable bool `yaml:"enable"`

	Sockc5 []string `yaml:"socks5"`
}
type Mysql struct {
	Ip       string `yaml:"ip"`
	Port     string `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}
type flagStruct struct {
	config_file string
}

var app appConfig
var robot Robots

const userAgent = `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36`

func init_config(flag flagStruct) {
	yamlFile, err := os.ReadFile(flag.config_file)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(yamlFile, &app)
	if err != nil {
		panic(err)
	}
	if !app.Exec.Enable.Search && !app.Exec.Enable.Seller && !app.Exec.Enable.Trn {
		panic("没有启动功能，检查配置文件的enable配置的选项")
	}
	log.Infof("程序标识:%d 主机标识:%d", app.Basic.App_id, app.Basic.Host_id)
}
func init_rebots() {
	robotTxt := fmt.Sprintf("https://%s/robots.txt", app.Domain)

	log.Infof("加载文件: %s", robotTxt)
	txt, err := request_get(robotTxt, userAgent)
	if err != nil {
		log.Error("网络错误")
		panic(err)
	}
	robot = GetRobotFromTxt(txt)
}
func init_mysql() {
	DB, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", app.Mysql.Username, app.Mysql.Password, app.Mysql.Ip, app.Mysql.Port, app.Mysql.Database))
	if err != nil {
		panic(err)
	}
	DB.SetConnMaxLifetime(100)
	DB.SetMaxIdleConns(10)
	if err := DB.Ping(); err != nil {
		panic(err)
	}
	log.Info("数据库已连接")
	app.db = DB
}
func init_network() {
	log.Info("网络测试开始")

	var s search
	s.en_key = "Hardware+electrician"
	_, err := s.request(0)
	if err != nil {
		log.Error("网络错误")
		panic(err)
	}

}
func init_signal() {
	// 创建一个通道来接收操作系统的信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGABRT)

	go func() {
		<-sigCh
		log.Info("")
		log.Infof("程序即将结束")
		app.end()
		app.db.Close()
		log.Infof("程序结束")
		os.Exit(0)
	}()
}
func init_flag() flagStruct {
	var f flagStruct
	flag.StringVar(&f.config_file, "c", "config.yaml", "打开配置文件")
	flag.Parse()
	return f
}

func main() {
	f := init_flag()
	init_config(f)
	init_rebots()
	init_mysql()
	init_network()
	init_signal()

	app.start()
	for {
		var s search
		s.main()

		var seller sellerStruct
		seller.main()

		var trn trnStruct
		trn.main()
	}

}
func (app *appConfig) get_cookie() (string, error) {
	var cookie string
	if app.Basic.Host_id == 0 {
		return "", fmt.Errorf("配置文件中host_id为0，cookie将为空")
	}

	if err := app.db.QueryRow("select cookie from cookie where host_id = ?", app.Basic.Host_id).Scan(&cookie); err != nil {
		return "", err
	}
	cookie = strings.TrimSpace(cookie)
	if app.cookie != cookie {
		log.Infof("使用新cookie: %s", cookie)
	}

	app.cookie = cookie
	return app.cookie, nil
}
func (app *appConfig) start() {
	if app.Basic.Test {
		log.Infof("测试模式启动")
		return
	}
	r, err := app.db.Exec("insert into application (app_id) values(?)", app.Basic.App_id)
	if err != nil {
		panic(err)
	}
	id, err := r.LastInsertId()
	if err != nil {
		panic(err)
	}
	app.primary_id = id
}
func (app *appConfig) update(status int) {
	_, err := app.db.Exec("update application set status=? where id=?", status, app.primary_id)
	if err != nil {
		panic(err)
	}
}
func (app *appConfig) end() {
	if app.Basic.Test {
		return
	}
	if _, err := app.db.Exec("update application set status=? where id=?", MYSQL_APPLICATION_STATUS_OVER, app.primary_id); err != nil {
		log.Error(err)
	}
}
