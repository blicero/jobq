// /home/krylon/go/src/github.com/blicero/jobq/cli/cli.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 08. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-08-09 19:40:07 krylon>

// Package cli implements the user-facing side of the application.
package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/blicero/jobq/common"
	"github.com/blicero/jobq/logdomain"
	"github.com/blicero/jobq/monitor"
	"github.com/blicero/jobq/monitor/request"
)

func socketPath(queueName string) string {
	return fmt.Sprintf("/tmp/jobq.%s.%s.socket",
		os.Getenv("USER"),
		queueName)
}

// CLI provides the terminal based user interface of the application.
type CLI struct {
	log  *log.Logger
	conn *net.UnixConn
	addr net.UnixAddr
}

// Create creates a new CLI instance which connects to the given socket.
func Create() (*CLI, error) {
	var (
		err   error
		shell = new(CLI)
	)

	// We cannot connect before parsing the command line arguments, obviously.
	if shell.log, err = common.GetLogger(logdomain.CLI); err != nil {
		return nil, err
	} /* else if shell.conn, err = net.DialUnix(common.NetName, nil, &addr); err != nil {
		shell.log.Printf("[ERROR] Cannot connect to Unix socket %s: %s\n",
			path,
			err.Error())
		return nil, err
	} */

	return shell, nil
} // func Create(path string) (*CLI, error)

func (c *CLI) Execute() {
	var (
		startServer, clean bool
		slots              int
		queueName          string
		err                error
	)

	flag.StringVar(&queueName, "name", "default", "Name of the job queue to use")
	flag.BoolVar(&startServer, "server", false, "Start the JobQ daemon.")
	flag.BoolVar(&clean, "clean", false, "clean up finished jobs")
	flag.IntVar(&slots, "slots", 1, "Number of jobs to run in parallel")

	flag.Parse()

	c.addr = net.UnixAddr{
		Net:  common.NetName,
		Name: socketPath(queueName),
	}

	if startServer {
		c.runMonitor(queueName, slots)
		return
	} else if err = c.connect(); err != nil {
		return
	}

	defer c.conn.Close() // nolint: errcheck

	if clean {
		// Later
	} else if len(flag.Args()) == 0 {
		c.displayQueue()
	}
} // func (c *CLI) Execute()

func (c *CLI) connect() error {
	var err error
	if c.conn, err = net.DialUnix(common.NetName, nil, &c.addr); err != nil {
		c.conn = nil
		c.log.Printf("[ERROR] Cannot connect to socket %s: %s\n",
			c.addr.Name,
			err.Error())
	}
	return err
} // func (c *CLI) connect() error

func (c *CLI) runMonitor(name string, slots int) {
	var (
		sock string
		err  error
		mon  *monitor.Monitor
	)

	sock = socketPath(name)

	if mon, err = monitor.Create(name, sock, slots); err != nil {
		c.log.Printf("[ERROR] Failed to create Monitor: %s\n",
			err.Error())
		return
	}

	mon.Start()
	defer os.Remove(sock)

	var sigQ = make(chan os.Signal, 1)
	var ticker = time.NewTicker(time.Second)

	signal.Notify(sigQ, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	for mon.Active() {
		select {
		case <-ticker.C:
		case sig := <-sigQ:
			c.log.Printf("[INFO] Quitting on signal %s\n",
				sig)
			break
		}
	}
} // func (c *CLI) runMonitor(name string, slots int)

func (c *CLI) displayQueue() {
	var (
		err            error
		msg            monitor.Message
		cnt            int
		res            monitor.Response
		sndbuf, rcvbuf []byte
	)

	rcvbuf = make([]byte, common.BufferSize)

	msg = monitor.Message{
		Timestamp: time.Now(),
		Request:   request.QueueQueryStatus.String(),
	}

	if sndbuf, err = json.Marshal(&msg); err != nil {
		c.log.Printf("[ERROR] Cannot serialize Message: %s\n",
			err.Error())
		return
	} else if _, err = c.conn.Write(sndbuf); err != nil {
		c.log.Printf("[ERROR] Failed to send via socket %s: %s\n",
			c.addr,
			err.Error())
		return
	} else if cnt, err = c.conn.Read(rcvbuf); err != nil {
		c.log.Printf("[ERROR] Failed to read from socket: %s\n",
			err.Error())
		return
	} else if err = json.Unmarshal(rcvbuf[:cnt], &res); err != nil {
		c.log.Printf("[ERROR] Cannot parse response: %s\n\n%s\n",
			err.Error(),
			rcvbuf[:cnt])
		return
	}

	const jobTmpl = "%6d %6d %3d %s\n"

	for _, j := range res.Jobs {
		var cmd = strings.Join(j.Cmd, " ")
		fmt.Printf(jobTmpl, j.ID, j.PID, j.ExitCode, cmd)
	}

	fmt.Println("")
} // func (c *CLI) displayQueue()

// func (c *CLI) Parse(s string) error {
// 	var (
// 		err    error
// 		tokens []string
// 	)

// 	if tokens, err = shlex.Split(s); err != nil {
// 		return err
// 	}

// } // func (c *CLI) Parse(s string) error
