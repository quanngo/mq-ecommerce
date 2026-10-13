package server

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"main/component/services"
	"main/internal"
	"main/utils"
	"os"
	"strings"
)

// RunCreateUser implements `go run main.go create-user`:
//
//	go run main.go create-user --email an.nguyen@mq.com --name "An Nguyen" --password-stdin < pw.txt
//	go run main.go create-user --email an.nguyen@mq.com --name "An Nguyen"   # Google-only account
//
// The password is only ever read from stdin, so it never lands in shell
// history or the process list.
func RunCreateUser(args []string) {
	email, name, password, err := parseCreateUserArgs(args, os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create-user:", err)
		os.Exit(2)
	}

	utils.AppConfig, err = initConfig()
	if err != nil {
		panic(err)
	}
	db := internal.NewPgDatabase().ConnectPgDatabase(utils.AppConfig.Database)
	authSvc := newAuthService(db, utils.AppConfig.Auth, nil)

	id, err := authSvc.CreateUser(context.Background(), services.CreateUserInput{Email: email, DisplayName: name, Password: password})
	if err != nil {
		fmt.Fprintln(os.Stderr, "create-user:", err)
		os.Exit(1)
	}
	kind := "google-only"
	if password != nil {
		kind = "password + google"
	}
	fmt.Printf("created app_user id=%d (%s)\n", id, kind)
}

func parseCreateUserArgs(args []string, stdin io.Reader) (email, name string, password *string, err error) {
	fs := flag.NewFlagSet("create-user", flag.ContinueOnError)
	fs.StringVar(&email, "email", "", "login email (required)")
	fs.StringVar(&name, "name", "", "display name")
	pwStdin := fs.Bool("password-stdin", false, "read the password from the first line of stdin")
	if err = fs.Parse(args); err != nil {
		return
	}
	if strings.TrimSpace(email) == "" {
		err = errors.New("--email is required")
		return
	}
	if *pwStdin {
		line, rerr := bufio.NewReader(stdin).ReadString('\n')
		if rerr != nil && rerr != io.EOF {
			err = rerr
			return
		}
		pw := strings.TrimRight(line, "\r\n")
		if pw == "" {
			err = errors.New("--password-stdin given but stdin was empty")
			return
		}
		password = &pw
	}
	return
}
