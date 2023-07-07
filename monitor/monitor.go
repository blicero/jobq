// /home/krylon/go/src/github.com/blicero/jobq/monitor/monitor.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-07 10:24:53 krylon>

// Package monitor is the nexus of the batch system.
package monitor

import (
	"log"
	"net"
	"sync"
	"sync/atomic"

	"github.com/blicero/jobq/common"
	"github.com/blicero/jobq/database"
	"github.com/blicero/jobq/logdomain"
	"github.com/blicero/jobq/queue"
)

type Monitor struct {
	log    *log.Logger
	db     *database.Database
	q      *queue.Queue
	active atomic.Bool
	lock   sync.RWMutex
	slots  int
	ctl    net.Conn
}

func Create(sock string, slots int) (*Monitor, error) {
	var (
		err error
		m   = &Monitor{
			slots: slots,
		}
	)

	if m.log, err = common.GetLogger(logdomain.Monitor); err != nil {
		return nil, err
	} else if m.db, err = database.Open(common.DbPath); err != nil {
		m.log.Printf("[ERROR] Cannot open database at %s: %s\n",
			common.DbPath,
			err.Error())
		return nil, err
	}

	return m, nil
} // func Create(sock string) (*Monitor, error)
