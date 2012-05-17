package cmd

import (
	"fmt"
	"io"
	"net/http"
)

type Manager struct {
	commands map[string]interface{}
	Stdout   io.Writer
	Stderr   io.Writer
}

func (m *Manager) Register(command interface{}) {
	if m.commands == nil {
		m.commands = make(map[string]interface{})
	}
	name := command.(Infoer).Info().Name
	_, found := m.commands[name]
	if found {
		panic(fmt.Sprintf("command already registered: %s", name))
	}
	m.commands[name] = command
}

func (m *Manager) Run(args []string) {
	if len(args) == 0 {
		args = []string{"help"}
	}
	command, exist := m.commands[args[0]]
	if !exist {
		io.WriteString(m.Stderr, fmt.Sprintf("command %s does not exist\n", args[0]))
		return
	}
	switch command.(type) {
	case CommandContainer:
		if len(args) > 1 {
			args = args[1:]
		}
		subcommand, exist := command.(CommandContainer).Subcommands()[args[0]]
		if exist {
			command = subcommand
		}
	}
	err := command.(Command).Run(&Context{args[1:], m.Stdout, m.Stderr}, NewClient(&http.Client{}))
	if err != nil {
		io.WriteString(m.Stderr, err.Error())
	}
}

func NewManager(stdout, stderr io.Writer) Manager {
	m := Manager{Stdout: stdout, Stderr: stderr}
	m.Register(&Help{manager: &m})
	return m
}

type CommandContainer interface {
	Subcommands() map[string]interface{}
}

type Infoer interface {
	Info() *Info
}

type Command interface {
	Run(context *Context, client Doer) error
}

type Context struct {
	Args   []string
	Stdout io.Writer
	Stderr io.Writer
}

type Info struct {
	Name    string
	MinArgs int
	Usage   string
	Desc    string
}

type Help struct {
	manager *Manager
}

func (c *Help) Info() *Info {
	return &Info{
		Name:  "help",
		Usage: "glb command [args]",
	}
}

func (c *Help) Run(context *Context, client Doer) error {
	output := ""
	if len(context.Args) > 0 {
		info := c.manager.commands[context.Args[0]].(Infoer).Info()
		output = output + fmt.Sprintf("Usage: %s\n", info.Usage)
		output = output + fmt.Sprintf("\n%s\n", info.Desc)
	} else {
		output = output + fmt.Sprintf("Usage: %s\n", c.Info().Usage)
	}
	io.WriteString(context.Stdout, output)
	return nil
}
