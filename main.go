package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"

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
	db         *sql.DB
	primary_id int64
}
type Identified struct {
	App int `yaml:"app"`
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

func init_signal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)
	<-c
	app.end()
	app.db.Close()
	log.Infof("程序结束")
}
func main() {
	init_config()
	app.start()

	// go init_signal()

	// var s search
	// s.main()

	var seller sellerStruct
	seller.main()

	var trn trnStruct
	trn.main()

	app.end()
}
func (app *appConfig) start() {
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
	app.db.Exec("update into application set status=? where id=?", MYSQL_APPLICATION_STATUS_OVER, app.primary_id)
}
