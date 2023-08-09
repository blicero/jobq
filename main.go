// /home/krylon/go/src/github.com/blicero/jobq/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 18. 06. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-08-09 20:53:55 krylon>

package main

import (
	"fmt"
	"os"

	"github.com/blicero/jobq/cli"
	"github.com/blicero/jobq/common"
)

func main() {
	fmt.Printf("%s %s, built on %s\n",
		common.AppName,
		common.Version,
		common.BuildStamp.Format(common.TimestampFormat))

	var (
		err   error
		shell *cli.CLI
	)

	if shell, err = cli.Create(); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to create CLI handler: %s\n",
			err.Error())
		os.Exit(1)
	}

	shell.Execute()
}
