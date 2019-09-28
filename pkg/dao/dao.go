package dao

import (
	"database/sql"
)

var Data  *sql.DB

func InitDB() error{
	db , err := sql.Open("sqlite3","./monitor.db")
	if err != nil {
		return err
	}

	Data = db
	return nil
}

func InsertOrUpdate() {

}

func Delete(){

}
