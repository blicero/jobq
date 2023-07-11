// /home/krylon/go/src/github.com/blicero/jobq/monitor/01_monitor_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-11 20:35:39 krylon>

package monitor

import (
	"fmt"
	"os"
	"testing"

	"github.com/blicero/jobq/job"
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

	socketPath = path

	if mon, err = Create(name, path, 1); err != nil {
		mon = nil
		t.Fatalf("Cannot create Monitor: %s", err.Error())
	}

	mon.Start()
} // func TestMonCreate(t *testing.T)

func TestMonSubmit(t *testing.T) {
	var (
		err error
		j   *job.Job
	)
} // func TestMonSubmit(t *testing.T)
