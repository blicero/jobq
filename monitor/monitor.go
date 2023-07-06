// /home/krylon/go/src/github.com/blicero/jobq/monitor/monitor.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-06 22:10:03 krylon>

// Package monitor is the nexus of the batch system.
package monitor

import (
	"log"
	"sync"

	"github.com/blicero/jobq/database"
	"github.com/blicero/jobq/queue"
)

type Monitor struct {
	log   *log.Logger
	db    *database.Database
	q     *queue.Queue
	lock  sync.RWMutex
	slots int
}
