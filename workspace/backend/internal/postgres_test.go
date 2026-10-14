package internal

import (
	"main/utils"
	"testing"
)

func TestPgConnString_QuotesValues(t *testing.T) {
	cases := map[string]struct{ password, want string }{
		"empty password": {"", `password='' sslmode='disable'`},
		"space":          {"pa ss", `password='pa ss' sslmode='disable'`},
		"quote":          {`it's`, `password='it\'s' sslmode='disable'`},
		"backslash":      {`a\b`, `password='a\\b' sslmode='disable'`},
		"looks like key": {"sslmode=require", `password='sslmode=require' sslmode='disable'`},
	}
	for name, tc := range cases {
		got := pgConnString(utils.PgDatabase{Host: "127.0.0.1", Port: 5432, User: "postgres", DbName: "mq", Password: tc.password, SslMode: "disable"})
		want := `host='127.0.0.1' port=5432 user='postgres' dbname='mq' ` + tc.want
		if got != want {
			t.Errorf("%s:\n got %s\nwant %s", name, got, want)
		}
	}
}
