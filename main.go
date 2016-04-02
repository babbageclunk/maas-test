// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package main

import (
	"fmt"
	"os"

	"github.com/juju/cmd"
	"github.com/juju/errors"
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

	fabrics, err := controller.Fabrics()
	if err != nil {
		return err
	}
	for _, fabric := range fabrics {
		fmt.Printf("Fabric %s(%d) has %d vlans\n", fabric.Name(), fabric.ID(), len(fabric.VLANs()))
	}

	for _, zone := range zones {
		fmt.Printf("Zone: %s (%s)\n", zone.Name(), zone.Description())
	}

	machines, err := controller.Machines(gomaasapi.MachinesArgs{})
	if err != nil {
		return err
	}

	for i, machine := range machines {
		fmt.Printf("\n-- machine %d\n", i+1)
		fmt.Printf("fqdn: %s\n", machine.FQDN())
		fmt.Printf("OS: %s/%s\n", machine.OperatingSystem(), machine.DistroSeries())
		fmt.Printf("Power: %s\n", machine.PowerState())
	}

	id := machines[0].SystemID()
	fmt.Printf("\nAsking for machine with system ID: %s\n", id)

	machines, err = controller.Machines(gomaasapi.MachinesArgs{
		SystemIds: []string{id},
	})
	if err != nil {
		return errors.Trace(err)
	}
	fmt.Printf("Should just have 1 result: %d\n", len(machines))
	fmt.Printf("%s\n\n", machines[0].SystemID())

	// Try to allocate a machine, dry run.

	// _, err = controller.AllocateMachine(gomaasapi.AllocateMachineArgs{
	// 	MinMemory: 2500,
	// 	DryRun:    true,
	// })
	// if err != nil {
	// 	fmt.Printf("Error allocating machine: %s\n", err.Error())
	// 	fmt.Printf("is bad request: %v\n", errors.IsBadRequest(err))
	// 	fmt.Printf("stackerrors.ErrorStack(err))
	// }

	return nil
}
