package app

import "github.com/jinzhu/gorm"

//Application the main application
type Application struct {
	db *gorm.DB
}

//NewApplication constructor
func New(db *gorm.DB) Application {
	return Application{
		db: db,
	}
}
