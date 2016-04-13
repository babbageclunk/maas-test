// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/gomaasapi"
	"github.com/juju/loggo"
	"launchpad.net/gnuflag"
)

var logger = loggo.GetLogger("maas-test")

func main() {
	rand.Seed(time.Now().UnixNano())
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
	read    bool
	debug   bool

	action string
	args   []string
	parent string
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
	f.StringVar(&c.parent, "parent", "", "a parent")
	f.BoolVar(&c.read, "read", false, "read the file first")
	f.BoolVar(&c.debug, "debug", false, "log at trace")
}

func (c *maasCommand) Init(args []string) error {
	if len(args) > 0 {
		c.action, c.args = args[0], args[1:]
	}
	return nil
}

func (c *maasCommand) Run(ctx *cmd.Context) error {
	if c.debug {
		loggo.GetLogger("").SetLogLevel(loggo.TRACE)
	} else {
		loggo.GetLogger("").SetLogLevel(loggo.INFO)
	}

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
	case "start":
		return c.start(controller)
	case "create-device":
		return c.createDevice(controller)
	case "delete-devices":
		return c.deleteDevices(controller)
	case "list-files":
		return c.listFiles(controller)
	case "add-file":
		return c.addFile(controller)
	case "read-file":
		return c.readFile(controller)
	case "delete-file":
		return c.deleteFile(controller)
	case "container":
		return c.container(controller)
	case "unlink-subnet":
		return c.unlinkSubnet(controller)
	default:
		fmt.Printf("unknown action: %q\n\n", c.action)
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
		SystemIDs: []string{id},
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

	machine, match, err := controller.AllocateMachine(gomaasapi.AllocateMachineArgs{
		Hostname: c.args[0],
	})
	if err != nil {
		return dumpErr(err)
	}
	fmt.Printf("match: %#v\n", match)
	fmt.Printf("Allocated machine: %s\n", machine.FQDN())
	return nil
}

func (c *maasCommand) release(controller gomaasapi.Controller) error {
	err := controller.ReleaseMachines(gomaasapi.ReleaseMachinesArgs{
		SystemIDs: c.args,
	})
	if err != nil {
		return dumpErr(err)
	}
	fmt.Printf("Released successfully\n")
	return nil

}

func (c *maasCommand) getMachine(controller gomaasapi.Controller, hostname string) (gomaasapi.Machine, error) {
	machines, err := controller.Machines(gomaasapi.MachinesArgs{
		Hostnames: []string{hostname},
	})
	if err != nil {
		return nil, err
	}

	if len(machines) != 1 {
		return nil, errors.Errorf("expected one result, got %d", len(machines))
	}
	return machines[0], nil
}

func (c *maasCommand) start(controller gomaasapi.Controller) error {

	if len(c.args) != 2 {
		return errors.Errorf("missing args: 'start <hostname> <series>'")
	}

	hostname := c.args[0]
	series := c.args[1]

	machine, err := c.getMachine(controller, hostname)
	if err != nil {
		return dumpErr(err)
	}

	err = machine.Start(gomaasapi.StartArgs{
		DistroSeries: series,
	})

	if err != nil {
		return dumpErr(err)
	}

	fmt.Printf("Started successfully\n")
	return nil
}

func (c *maasCommand) createDevice(controller gomaasapi.Controller) error {
	args := gomaasapi.CreateDeviceArgs{
		Parent: c.parent,
	}
	if len(c.args) > 0 {
		args.Hostname, args.MACAddresses = c.args[0], c.args[1:]
	}

	device, err := controller.CreateDevice(args)
	if err != nil {
		return dumpErr(err)
	}
	fmt.Printf("Device created: %s\n", device.SystemID())
	return nil

}

func (c *maasCommand) deleteDevices(controller gomaasapi.Controller) error {
	if len(c.args) < 1 {
		fmt.Println("expected <machine name>")
		return errors.New("missing args")
	}
	hostname := c.args[0]
	machine, err := c.getMachine(controller, hostname)
	if err != nil {
		return dumpErr(err)
	}

	devices, err := machine.Devices(gomaasapi.DevicesArgs{})
	if err != nil {
		return dumpErr(err)
	}

	for _, device := range devices {
		err := device.Delete()
		if err != nil {
			return dumpErr(err)
		}
		fmt.Printf("deleted device %q\n", device.Hostname())
	}
	return nil
}

func (c *maasCommand) listFiles(controller gomaasapi.Controller) error {
	prefix := ""
	switch count := len(c.args); {
	case count == 1:
		prefix = c.args[0]
	case count > 1:
		return errors.New("too many args")
	}
	files, err := controller.Files(prefix)
	if err != nil {
		return dumpErr(err)
	}
	for i, f := range files {
		fmt.Printf("%d: %s (%s)\n", i, f.Filename(), f.AnonymousURL())
	}
	return nil
}

func (c *maasCommand) addFile(controller gomaasapi.Controller) error {
	if len(c.args) != 2 {
		return errors.Errorf("expected <filename> <file path>")
	}
	filename, path := c.args[0], c.args[1]

	args := gomaasapi.AddFileArgs{
		Filename: filename,
	}
	if c.read {
		logger.Infof("reading content first")
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		args.Content = content
	} else {
		logger.Infof("opening file and providing reader")
		info, err := os.Stat(path)
		if err != nil {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		args.Reader = file
		args.Length = info.Size()
	}

	err := controller.AddFile(args)
	if err != nil {
		return dumpErr(err)
	}
	fmt.Printf("file added successfully\n")
	return nil
}

func (c *maasCommand) readFile(controller gomaasapi.Controller) error {
	if len(c.args) != 1 {
		return errors.Errorf("expected <filename>")
	}
	filename := c.args[0]
	var file gomaasapi.File

	if c.read {
		logger.Infof("Get file directly")
		read, err := controller.GetFile(filename)
		if err != nil {
			return dumpErr(err)
		}
		file = read
	} else {
		files, err := controller.Files(filename)
		if err != nil {
			return dumpErr(err)
		}
		for _, f := range files {
			if f.Filename() == filename {
				file = f
			}
		}
		if file == nil {
			return errors.New("file not found")
		}
	}

	content, err := file.ReadAll()
	if err != nil {
		return dumpErr(err)
	}
	fmt.Println(string(content))
	return nil
}

func (c *maasCommand) deleteFile(controller gomaasapi.Controller) error {
	if len(c.args) != 1 {
		return errors.Errorf("expected <filename>")
	}
	filename := c.args[0]
	file, err := controller.GetFile(filename)
	if err != nil {
		return dumpErr(err)
	}

	err = file.Delete()
	if err != nil {
		return dumpErr(err)
	}
	fmt.Printf("File %q deleted.\n", filename)
	return nil
}

func dumpErr(err error) error {
	fmt.Printf("\nError type: %T\n", errors.Cause(err))
	fmt.Println(errors.ErrorStack(err))
	return err
}

func (c *maasCommand) container(controller gomaasapi.Controller) error {
	if c.parent == "" {
		return errors.Errorf("missing parent")
	}

	machine, err := c.getMachine(controller, c.parent)
	if err != nil {
		return dumpErr(err)
	}
	link := machine.BootInterface().Links()[0]

	args := gomaasapi.CreateMachineDeviceArgs{
		InterfaceName: "eth1",
		MACAddress:    newMACAddress(),
		Subnet:        link.Subnet(),
	}
	if len(c.args) > 0 {
		args.Hostname = c.args[0]
	}
	device, err := machine.CreateDevice(args)
	if err != nil {
		return dumpErr(err)
	}
	fmt.Printf("Device %q created\n", device.Hostname())

	return nil
}

func (c *maasCommand) unlinkSubnet(controller gomaasapi.Controller) error {
	if len(c.args) < 3 {
		fmt.Println("expected <device name> <interface name> <subnet name>")
		return errors.New("missing args")
	}
	deviceName := c.args[0]
	interfaceName := c.args[1]
	subnetName := c.args[2]
	devices, err := controller.Devices(gomaasapi.DevicesArgs{
		Hostname: []string{deviceName},
	})
	if err != nil {
		return dumpErr(err)
	}

	if len(devices) == 0 {
		return errors.Errorf("device name %q not found", deviceName)
	} else if len(devices) > 1 {
		return errors.Errorf("wat? %#v", devices)
	}
	device := devices[0]

	ifaces := device.InterfaceSet()

	var iface gomaasapi.Interface
	for _, i := range ifaces {
		if i.Name() == interfaceName {
			iface = i
			break
		}
	}
	if iface == nil {
		return errors.Errorf("interfae name %q not found", interfaceName)
	}

	for _, link := range iface.Links() {
		if link.Subnet() != nil && link.Subnet().Name() == subnetName {
			err = iface.UnlinkSubnet(link.Subnet())
			if err != nil {
				return dumpErr(err)
			}
			fmt.Printf("subnet %q unlinked from %s interface %s\n", subnetName, device.Hostname(), interfaceName)
			return nil
		}
	}
	return errors.Errorf("subnet %q not linked to %s interface %s", subnetName, device.Hostname(), interfaceName)
}

func newMACAddress() string {
	var values []string
	for i := 0; i < 6; i++ {
		values = append(values, fmt.Sprintf("%02x", rand.Intn(256)))
	}
	return strings.Join(values, "-")
}
