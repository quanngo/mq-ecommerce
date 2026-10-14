package internal

import (
	"fmt"
	"main/utils"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type pgDatabase struct{}

func NewPgDatabase() *pgDatabase {
	return &pgDatabase{}
}

func (db *pgDatabase) ConnectPgDatabase(p utils.PgDatabase) *sqlx.DB {
	strConn := pgConnString(p)
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

// pgConnString quotes every value, so an empty password or one containing a
// space or quote can't swallow the next key (unquoted, "password= sslmode=x"
// reads as the password "sslmode=x").
func pgConnString(p utils.PgDatabase) string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=%s",
		pgQuote(p.Host), p.Port, pgQuote(p.User), pgQuote(p.DbName), pgQuote(p.Password), pgQuote(p.SslMode))
}

// pgQuote writes a libpq keyword value: single-quoted, with \ and ' escaped.
func pgQuote(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(s) + "'"
}
