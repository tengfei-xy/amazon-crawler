package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/tengfei-xy/go-log"
	"gopkg.in/yaml.v3"
)

const AMAZON_UK = "https://www.amazon.co.uk"
const MYSQL_APPLICATION_STATUS_START int = 0
const MYSQL_APPLICATION_STATUS_OVER int = 1
const MYSQL_APPLICATION_STATUS_SEARCH int = 2
const MYSQL_APPLICATION_STATUS_SELLER int = 3
const MYSQL_APPLICATION_STATUS_TRN int = 4

type appConfig struct {
	Mysql      `yaml:"mysql"`
	Identified `yaml:"identified"`
	Proxy      `yaml:"proxy"`
	db         *sql.DB
	primary_id int64
}
type Identified struct {
	App  int  `yaml:"app"`
	Test bool `yaml:"test"`
}
type Proxy struct {
	Sockc5 []string `yaml:"socks5"`
}
type Mysql struct {
	Ip       string `yaml:"ip"`
	Port     string `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

var app appConfig

func init_config() {
	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(yamlFile, &app)
	if err != nil {
		panic(err)
	}
	log.Infof("程序标识:%d", app.Identified.App)

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
	_, err := s.NewRequest(0)
	if err != nil {
		log.Error("网络错误")
		panic(err)
	}

	log.Info("网页测试成功")

}
func init_signal() {
	// 创建一个通道来接收操作系统的信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGABRT)

	go func() {
		<-sigCh
		cleanup()
	}()
}
func main() {
	init_config()
	init_network()
	init_mysql()
	init_signal()
	runApplication()

}
func cleanup() {
	log.Infof("\n程序即将结束")
	app.end()
	app.db.Close()
	log.Infof("程序结束")
	os.Exit(0)
}
func runApplication() {

	app.start()

	var s search
	s.main()

	var seller sellerStruct
	seller.main()

	var trn trnStruct
	trn.main()
}
func (app *appConfig) start() {
	if app.Identified.Test {
		log.Infof("测试模式启动")
		return
	}
	r, err := app.db.Exec("insert into application (app_id) values(?)", app.Identified.App)
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
	if app.Identified.Test {
		return
	}
	app.db.Exec("update into application set status=? where id=?", MYSQL_APPLICATION_STATUS_OVER, app.primary_id)
}

// 随机挂起 x 秒
func sleep(i int) {
	time.Sleep(time.Duration(i) * time.Second)
}
