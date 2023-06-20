package easyserver

import (
	"log"

	"github.com/jmoiron/sqlx"
)

func NewDB(address string, poolNum int) *sqlx.DB {
	db, err := sqlx.Open("mysql", address)
	if err != nil {
		log.Fatalln(err)
	}
	db.SetMaxOpenConns(poolNum)
	db.SetMaxIdleConns(poolNum)
	return db
}
