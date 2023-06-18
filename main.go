// /home/krylon/go/src/github.com/blicero/jobq/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 18. 06. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-06-18 20:32:43 krylon>

package main

import (
	"fmt"

	"github.com/blicero/jobq/common"
)

func main() {
	fmt.Printf("%s %s, built on %s",
		common.AppName,
		common.Version,
		common.BuildStamp.Format(common.TimestampFormat))
}
