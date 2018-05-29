package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/jinzhu/gorm"

	"ivorareteambot/app"
	"ivorareteambot/config"
	"ivorareteambot/controller"
	"ivorareteambot/types"
)

import (
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

const dbLoggingEnabled = true

var db *gorm.DB
var taskTitle string
var currentTask types.Task

func GetFunctionName() string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return fmt.Sprintf("%s:%d", frame.Line, frame.Function)
}

func main() {
	cfg := config.GetConfig()

	var defCfg config.Config
	if cfg.Slack.InToken == defCfg.Slack.InToken {
		log.Println("SlackInToken configuration field is not set. Please set it in configuration file «config/config.yml».")
	}
	if cfg.Slack.OutToken == defCfg.Slack.OutToken {
		log.Println("SlackOutToken configuration field is not set. Please set it in configuration file «config/config.yml».")
	}

	db, dbErr := openDB("mysql", cfg)
	if dbErr != nil {
		panic(dbErr)
	}
	defer db.Close()

	a := app.New(db)
	c := controller.New(
		a,
		cfg.Slack.InToken,
		cfg.Slack.OutToken,
	)

	c.InitRouters()

	c.Serve()
}
func openDB(dialect string, config config.Config) (*gorm.DB, error) {
	var dsn = fmt.Sprintf(
		"%s:%s@tcp(%s:%v)/%s?charset=%s&parseTime=%s&loc=Local",
		config.DB.User,
		config.DB.Password,
		config.DB.Host,
		config.DB.Port,
		config.DB.Name,
		config.DB.Charset,
		config.DB.ParseTime,
	)

	log.Printf("dsn is:\n%+v", dsn)

	db, err := gorm.Open(dialect, dsn)
	db.LogMode(dbLoggingEnabled)

	return db, err
}
