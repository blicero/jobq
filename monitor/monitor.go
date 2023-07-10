// /home/krylon/go/src/github.com/blicero/jobq/monitor/monitor.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-10 11:24:58 krylon>

// Package monitor is the nexus of the batch system.
package monitor

import (
	"encoding/json"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/blicero/jobq/common"
	"github.com/blicero/jobq/database"
	"github.com/blicero/jobq/logdomain"
	"github.com/blicero/jobq/queue"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/shlex"
)

// Monitor runs the Job Queue and accepts requests from client.
type Monitor struct {
	name   string
	path   string
	log    *log.Logger
	db     *database.Database
	q      *queue.Queue
	active bool
	lock   sync.RWMutex
	slots  int
	ctl    net.Conn
}

// Create creates and returns a new Monitor.
func Create(name, string, sock string, slots int) (*Monitor, error) {
	var (
		err error
		m   = &Monitor{
			name:  name,
			path:  sock,
			slots: slots,
		}
		addr = net.UnixAddr{
			Name: sock,
			Net:  "unixgram",
		}
	)

	if m.log, err = common.GetLogger(logdomain.Monitor); err != nil {
		return nil, err
	} else if m.db, err = database.Open(common.DbPath); err != nil {
		m.log.Printf("[ERROR] Cannot open database at %s: %s\n",
			common.DbPath,
			err.Error())
		return nil, err
	} else if m.q, err = queue.New(); err != nil {
		m.log.Printf("[ERROR] Cannot create new Job Queue: %s",
			err.Error())
		return nil, err
	} else if m.ctl, err = net.ListenUnixgram("unixgram", &addr); err != nil {
		m.log.Printf("[ERROR] Cannot open control socket %s: %s\n",
			sock,
			err.Error())
		return nil, err
	}

	return m, nil
} // func Create(sock string) (*Monitor, error)

// Start starts the Monitor and its components.
func (m *Monitor) Start() {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.active {
		m.log.Printf("[ERROR] Monitor is already active.\n")
		return
	}

	m.active = true
	m.q.Start()

	go m.loop()
} // func (m *Monitor) Start()

// Stop tells the Monitor to stop.
func (m *Monitor) Stop() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.q.Stop()
	m.active = false
} // func (m *Monitor) Stop()

// Active returns the Monitor's active flag
func (m *Monitor) Active() bool {
	m.lock.RLock()
	var active = m.active
	m.lock.RUnlock()
	return active
} // func (m *Monitor) Active() bool

func (m *Monitor) loop() {
	defer m.q.Stop()

	var buffer = make([]byte, 65536)

	for m.Active() {
		var (
			err error
			cnt int
			msg Message
		)

		if cnt, err = m.ctl.Read(buffer); err != nil {
			m.log.Printf("[ERROR] Cannot read from socket: %s\n",
				err.Error())
		} else if err = json.Unmarshal(buffer[:cnt], &msg); err != nil {
			m.log.Printf("[ERROR] Cannot decode JSON message: %s\n%s\n\n",
				err.Error(),
				buffer[:cnt])
		}

		m.log.Printf("[DEBUG] Got one message: %s\n",
			msg.Request)
		go m.handleMessage(msg)
	}
} // func (m *Monitor) loop()

func (m *Monitor) handleMessage(msg Message) {
	m.log.Printf("[DEBUG] Handle message: %s\n",
		spew.Sdump(&msg))

	var (
		err error
		req []string
		cmd string
	)

	if req, err = shlex.Split(msg.Request); err != nil {
		m.log.Printf("[ERROR] Cannot parse Request: %s\nRaw: %s\n",
			err.Error(),
			msg.Request)
		return
	}

	cmd = strings.ToLower(req[0])

	switch cmd {
	default:
		m.log.Printf("[ERROR] Dont' know how to handle %s\n",
			cmd)
		return
	}
} // func (m *Monitor) handleMessage(msg *Message)
