// /home/krylon/go/src/github.com/blicero/jobq/monitor/01_monitor_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-10 20:14:13 krylon>

package monitor

import (
	"fmt"
	"os"
	"testing"
)

var mon *Monitor

func TestMonCreate(t *testing.T) {
	const name = "TestMonitor"

	var (
		err  error
		path = fmt.Sprintf("/tmp/jobq.%s.%s",
			os.Getenv("USER"),
			name)
	)

	if mon, err = Create(name, path, 1); err != nil {
		mon = nil
		t.Fatalf("Cannot create Monitor: %s", err.Error())
	}

	mon.Start()
} // func TestMonCreate(t *testing.T)
