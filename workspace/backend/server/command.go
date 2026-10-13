package server

import (
	"os"

	"github.com/sirupsen/logrus"
)

type Command struct {
	Script      string
	Desc        string
	ExecuteFunc func(args []string)
}

type CommandManager struct {
	commands []Command
}

// AddCommand add new app command
func (c *CommandManager) AddCommand(command Command) {
	c.commands = append(c.commands, command)
}

// Execute exec command base on arguments
func (c *CommandManager) Execute() {
	args := os.Args
	commandStr := ""
	if len(args) >= 2 {
		commandStr = args[1]
	}

	for _, command := range c.commands {
		if command.Script == commandStr {
			logrus.Infof("[App] Running command: %v", command.Script)
			logrus.Infof("[App] Command description: %v", command.Desc)
			command.ExecuteFunc(args)
			return
		}
	}

	panic("No command to execute")
}
