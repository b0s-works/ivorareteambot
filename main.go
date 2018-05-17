package main

import (
	"runtime"
	"fmt"
	"github.com/jinzhu/gorm"
	"ivorareteambot/types"
	"ivorareteambot/config"
	"ivorareteambot/app"
	"ivorareteambot/controller"
	"net/http"
	"log"
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
	config := config.GetConfig();

	db, dbErr := openDB("mysql", config)
	if dbErr != nil {
		panic(dbErr)
	}
	defer db.Close()

	a := app.New(db)
	c := controller.New(
		a,
		config.SlackToken,
	)

	c.InitRouters()

	httpPort := 80
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	fmt.Printf("listening on %v\n", httpPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil))
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

	fmt.Printf("dsn: %+v", dsn)

	db, err := gorm.Open( dialect, dsn )
	db.LogMode( dbLoggingEnabled )

	return db, err
}