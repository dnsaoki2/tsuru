// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"github.com/globocom/commandmocker"
	"github.com/globocom/tsuru/api/app"
	"github.com/globocom/tsuru/api/bind"
	"github.com/globocom/tsuru/db"
	"github.com/globocom/tsuru/log"
	"github.com/globocom/tsuru/queue"
	"labix.org/v2/mgo/bson"
	. "launchpad.net/gocheck"
	stdlog "log"
	"strings"
	"time"
)

func (s *S) TestHandleMessages(c *C) {
	handler := MessageHandler{}
	err := handler.start()
	c.Assert(err, IsNil)
	defer handler.stop()
	a := app.App{
		Name: "nemesis",
		Units: []app.Unit{
			{
				Name:              "nemesis/0",
				MachineAgentState: "running",
				AgentState:        "started",
				InstanceState:     "running",
				Machine:           19,
			},
		},
		Env: map[string]bind.EnvVar{
			"http_proxy": {
				Name:   "http_proxy",
				Value:  "http://myproxy.com:3128/",
				Public: true,
			},
		},
		State: "started",
	}
	err = db.Session.Apps().Insert(a)
	c.Assert(err, IsNil)
	defer db.Session.Apps().Remove(bson.M{"name": a.Name})
	tmpdir, err := commandmocker.Add("juju", "$*")
	c.Assert(err, IsNil)
	defer commandmocker.Remove(tmpdir)
	messages, _, err := queue.Dial(handler.server.Addr())
	c.Assert(err, IsNil)
	messages <- queue.Message{Action: app.RegenerateApprc, Args: []string{a.Name}}
	time.Sleep(1e9)
	c.Assert(commandmocker.Ran(tmpdir), Equals, true)
	output := strings.Replace(commandmocker.Output(tmpdir), "\n", " ", -1)
	outputRegexp := `^.*19 cat > /home/application/apprc <<END # generated by tsuru.*`
	outputRegexp += `export http_proxy="http://myproxy.com:3128/" END $`
	c.Assert(output, Matches, outputRegexp)
}

func (s *S) TestHandleMessageErrors(c *C) {
	var data = []struct {
		action      string
		appName     string
		expectedLog string
	}{
		{
			action:      "unknown-action",
			appName:     "does not matter",
			expectedLog: `Error handling "unknown-action": invalid action.`,
		},
		{
			action:  app.RegenerateApprc,
			appName: "nemesis",
			expectedLog: `Error handling "regenerate-apprc" for the app "nemesis":` +
				` The status of the app should be "started", but it is "pending".`,
		},
		{
			action:      app.RegenerateApprc,
			appName:     "unknown-app",
			expectedLog: `Error handling "regenerate-apprc": app "unknown-app" does not exist.`,
		},
		{
			action:      app.RegenerateApprc,
			expectedLog: `Error handling "regenerate-apprc": this action requires at least 1 argument.`,
		},
	}
	var buf bytes.Buffer
	a := app.App{Name: "nemesis", State: "pending"}
	err := db.Session.Apps().Insert(a)
	c.Assert(err, IsNil)
	defer db.Session.Apps().Remove(bson.M{"name": a.Name})
	log.SetLogger(stdlog.New(&buf, "", 0))
	handler := MessageHandler{}
	handler.start()
	defer handler.stop()
	for _, d := range data {
		message := queue.Message{
			Action: d.action,
		}
		if d.appName != "" {
			message.Args = append(message.Args, d.appName)
		}
		handler.handle(message)
	}
	content := buf.String()
	lines := strings.Split(content, "\n")
	for i, d := range data {
		if lines[i] != d.expectedLog {
			c.Errorf("\nWant: %q.\nGot: %q", d.expectedLog, lines[i])
		}
	}
}
