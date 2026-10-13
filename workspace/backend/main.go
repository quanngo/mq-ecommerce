package main

import (
	"main/server"

	_ "main/docs"
)

// @title           MQ Document API
// @version         1.0
// @description     This is a sample document for API.
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
