package main

import (
	"main/server"

	_ "main/docs"
)

// @title           MQ E-commerce Admin API
// @version         1.0
// @description     Backend of the MQ e-commerce admin app (marketplace order sync, login, users, orders).
// @description
// @description     Every JSON endpoint answers HTTP 200 with the envelope {code, message, data}: `code` is the real status, `message` the error code on failure (e.g. UNAUTHENTICATED, FORBIDDEN), and `data` is an array whose first element is the documented response model.
// @description
// @description     Auth: sign in with POST /v1/auth/login (or the Google flow), which sets the httpOnly `ecom_session` cookie; send it with every request. State-changing calls must come from the frontend origin (Origin header checked). Required permissions per route: see workspace/backend/API_DOCUMENT.md.
// @termsOfService  http://swagger.io/terms/

// @contact.name   Quan Ngo
// @contact.email  ngonguyenminhquan@gmail.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:9091
// @BasePath  /api

func main() {
	cm := server.CommandManager{}

	cm.AddCommand(server.Command{
		Script: "",
		Desc:   "Run backend server",
		ExecuteFunc: func(args []string) {
			// Init job
			workerIns := server.InitializeWorkers()
			workerIns.RunCronJob()

			apiServer := server.Initialize()
			apiServer.Start()
		},
	})

	cm.AddCommand(server.Command{
		Script: "run-job",
		Desc:   "Run cronjob",
		ExecuteFunc: func(args []string) {
			server.InitializeJob()
		},
	})

	cm.AddCommand(server.Command{
		Script: "create-user",
		Desc:   "Create an admin-app login user: create-user --email a@mq.com --name \"A\" [--password-stdin]",
		ExecuteFunc: func(args []string) {
			server.RunCreateUser(args[2:])
		},
	})

	cm.Execute()
}
