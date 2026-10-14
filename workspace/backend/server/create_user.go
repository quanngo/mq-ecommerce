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
//	go run main.go create-user --email an.nguyen@mq.com --name "An Nguyen" --role admin --password-stdin < pw.txt
//	go run main.go create-user --email an.nguyen@mq.com --name "An Nguyen"   # Google-only, auth.default-role
//
// The password is only ever read from stdin, so it never lands in shell
// history or the process list. Roles: admin | operator | viewer (migrations/004).
func RunCreateUser(args []string) {
	in, err := parseCreateUserArgs(args, os.Stdin)
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

	id, err := authSvc.CreateUser(context.Background(), in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create-user:", err)
		os.Exit(1)
	}
	kind := "google-only"
	if in.Password != nil {
		kind = "password + google"
	}
	role := in.Role
	if role == "" {
		role = utils.AppConfig.Auth.WithDefaults().DefaultRole
	}
	fmt.Printf("created app_user id=%d role=%s (%s)\n", id, role, kind)
}

func parseCreateUserArgs(args []string, stdin io.Reader) (services.CreateUserInput, error) {
	var in services.CreateUserInput
	fs := flag.NewFlagSet("create-user", flag.ContinueOnError)
	fs.StringVar(&in.Email, "email", "", "login email (required)")
	fs.StringVar(&in.DisplayName, "name", "", "display name")
	fs.StringVar(&in.Role, "role", "", "role code: admin | operator | viewer (default: auth.default-role)")
	pwStdin := fs.Bool("password-stdin", false, "read the password from the first line of stdin")
	if err := fs.Parse(args); err != nil {
		return in, err
	}
	if strings.TrimSpace(in.Email) == "" {
		return in, errors.New("--email is required")
	}
	if *pwStdin {
		line, err := bufio.NewReader(stdin).ReadString('\n')
		if err != nil && err != io.EOF {
			return in, err
		}
		pw := strings.TrimRight(line, "\r\n")
		if pw == "" {
			return in, errors.New("--password-stdin given but stdin was empty")
		}
		in.Password = &pw
	}
	return in, nil
}
