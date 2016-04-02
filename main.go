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

	action string
	args   []string
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
	if len(args) > 0 {
		c.action, c.args = args[0], args[1:]
	}
	return nil
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

	switch c.action {
	case "":
		return c.noAction(controller)
	case "allocate":
		return c.allocate(controller)
	case "release":
		return c.release(controller)
	default:
		fmt.Printf("unknown action: %q", c.action)
	}

	return nil
}

func (c *maasCommand) noAction(controller gomaasapi.Controller) error {
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
		fmt.Printf("system id: %s\n", machine.SystemID())
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
	return nil
}

func (c *maasCommand) allocate(controller gomaasapi.Controller) error {
	// Try to allocate a machine, dry run.
	if len(c.args) != 1 {
		return errors.Errorf("Expected only one arg to allocate, got %#v", c.args)
	}

	machine, err := controller.AllocateMachine(gomaasapi.AllocateMachineArgs{
		Hostname: c.args[0],
	})
	if err != nil {
		fmt.Println(errors.ErrorStack(err))
		return err
	}

	fmt.Printf("Allocated machine: %s\n", machine.FQDN())
	return nil
}

func (c *maasCommand) release(controller gomaasapi.Controller) error {
	err := controller.ReleaseMachines(gomaasapi.ReleaseMachinesArgs{
		SystemIDs: c.args,
	})
	if err != nil {
		fmt.Println(errors.ErrorStack(err))
		return err
	}
	fmt.Printf("Released successfully\n")
	return nil

}
