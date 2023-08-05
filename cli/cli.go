// /home/krylon/go/src/github.com/blicero/jobq/cli/cli.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 08. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-08-05 23:20:40 krylon>

// Package cli implements the user-facing side of the application.
package cli

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/blicero/jobq/common"
	"github.com/blicero/jobq/logdomain"
)

// CLI provides the terminal based user interface of the application.
type CLI struct {
	log  *log.Logger
	conn *net.UnixConn
}

// Create creates a new CLI instance which connects to the given socket.
func Create(path string) (*CLI, error) {
	var (
		err  error
		addr = net.UnixAddr{
			Net:  common.NetName,
			Name: path,
		}
		shell = new(CLI)
	)

	if shell.log, err = common.GetLogger(logdomain.CLI); err != nil {
		return nil, err
	} else if shell.conn, err = net.DialUnix(common.NetName, nil, &addr); err != nil {
		shell.log.Printf("[ERROR] Cannot connect to Unix socket %s: %s\n",
			path,
			err.Error())
		return nil, err
	}

	return shell, nil
} // func Create(path string) (*CLI, error)

func (c *CLI) Execute() {
	var (
		startServer bool
		socketName  string
	)

	flag.BoolVar(&startServer, "server", false, "Start the JobQ daemon.")
	flag.StringVar(&socketName, "socket", fmt.Sprintf("/tmp/jobq.%s.socket", os.Getenv("USER")), "Path to the server socket")

	flag.Parse()

	if startServer {

	}
}

// func (c *CLI) Parse(s string) error {
// 	var (
// 		err    error
// 		tokens []string
// 	)

// 	if tokens, err = shlex.Split(s); err != nil {
// 		return err
// 	}

// } // func (c *CLI) Parse(s string) error
