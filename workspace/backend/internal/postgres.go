package internal

import (
	"fmt"
	"main/utils"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type pgDatabase struct{}

func NewPgDatabase() *pgDatabase {
	return &pgDatabase{}
}

func (db *pgDatabase) ConnectPgDatabase(p utils.PgDatabase) *sqlx.DB {
	strConn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=%s",
		p.Host, p.Port, p.User, p.DbName, p.Password, p.SslMode)
	var err error
	dbInstance, err := sqlx.Open("postgres", strConn)
	dbInstance.SetMaxIdleConns(p.MaxIdleConn)
	dbInstance.SetMaxOpenConns(p.MaxOpenConn)
	if err != nil {
		panic(err)
	}

	err = dbInstance.Ping()
	if err != nil {
		panic(err)
	}

	utils.ShowInfoLogs("Connect to postgres database %s:%d/%s successful", p.Host, p.Port, p.DbName)
	return dbInstance
}
