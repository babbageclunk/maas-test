// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package main

import (
	"fmt"
	"os"

	"github.com/juju/cmd"
	"github.com/juju/gomaasapi"
	"github.com/juju/loggo"
	"launchpad.net/gnuflag"
)

var logger = loggo.GetLogger("juju")

func main() {
	ctx, err := cmd.DefaultContext()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	os.Exit(cmd.Main(&maasCommand{}, ctx, os.Args[1:]))
}

type maasCommand struct {
	cmd.CommandBase

	baseurl string
	creds   string
}

func (c *maasCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "maas-test",
		Args:    "<basedir> <creds>",
		Purpose: "test maas 2.0",
	}
}

func (c *maasCommand) SetFlags(f *gnuflag.FlagSet) {
	f.StringVar(&c.baseurl, "base-url", "http://192.168.100.2/MAAS", "maas to test")
	f.StringVar(&c.creds, "creds", "", "maas oauth creds")
}

func (c *maasCommand) Init(args []string) error {
	return cmd.CheckEmpty(args)
}

func (c *maasCommand) Run(ctx *cmd.Context) error {
	loggo.GetLogger("maas").SetLogLevel(loggo.TRACE)

	controller, err := gomaasapi.NewController(gomaasapi.ControllerArgs{
		BaseURL: c.baseurl,
		APIKey:  c.creds,
	})

	if err != nil {
		return err
	}

	zones, err := controller.Zones()
	if err != nil {
		return err
	}

	for _, zone := range zones {
		fmt.Printf("Zone: %s (%s)\n", zone.Name(), zone.Description())
	}

	return nil
}
