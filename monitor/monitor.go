// /home/krylon/go/src/github.com/blicero/jobq/monitor/monitor.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-11 19:30:23 krylon>

// Package monitor is the nexus of the batch system.
package monitor

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/blicero/jobq/common"
	"github.com/blicero/jobq/database"
	"github.com/blicero/jobq/job"
	"github.com/blicero/jobq/logdomain"
	"github.com/blicero/jobq/monitor/request"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/shlex"
)

const minDbCnt = 4

// Monitor runs the Job Queue and accepts requests from client.
type Monitor struct {
	name   string
	path   string
	log    *log.Logger
	pool   *database.Pool
	active atomic.Bool
	slots  int
	ctl    *net.UnixConn
	seqCnt atomic.Int64
}

// Create creates and returns a new Monitor.
func Create(name, sock string, slots int) (*Monitor, error) {
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
	} else if m.pool, err = database.NewPool(minDbCnt); err != nil {
		m.log.Printf("[ERROR] Cannot open database at %s: %s\n",
			common.DbPath,
			err.Error())
		return nil, err
	} else if m.ctl, err = net.ListenUnixgram("unixgram", &addr); err != nil {
		m.log.Printf("[ERROR] Cannot open control socket %s: %s\n",
			sock,
			err.Error())
		return nil, err
	}

	return m, nil
} // func Create(name, sock string, slots int) (*Monitor, error)

// Start starts the Monitor and its components.
func (m *Monitor) Start() {
	if m.active.Load() {
		m.log.Printf("[ERROR] Monitor is already active.\n")
		return
	}

	m.active.Store(true)

	go m.ctlLoop()
	go m.jobLoop()
} // func (m *Monitor) Start()

// Stop tells the Monitor to stop.
func (m *Monitor) Stop() {
	m.active.Store(false)
} // func (m *Monitor) Stop()

// Active returns the Monitor's active flag
func (m *Monitor) Active() bool {
	return m.active.Load()
} // func (m *Monitor) Active() bool

func (m *Monitor) ctlLoop() {
	var buffer = make([]byte, 65536)

	for m.active.Load() {
		var (
			err  error
			cnt  int
			msg  Message
			addr *net.UnixAddr
		)

		// I should probably set a timeout for reading, so I can check
		// the active flag periodically?

		if cnt, addr, err = m.ctl.ReadFromUnix(buffer); err != nil {
			m.log.Printf("[ERROR] Cannot read from socket: %s\n",
				err.Error())
		} else if err = json.Unmarshal(buffer[:cnt], &msg); err != nil {
			m.log.Printf("[ERROR] Cannot decode JSON message: %s\n%s\n\n",
				err.Error(),
				buffer[:cnt])
		}

		m.log.Printf("[DEBUG] Got one message from %s: %s\n",
			addr,
			msg.Request)
		go m.handleMessage(msg, addr)
	}
} // func (m *Monitor) ctlLoop()

func (m *Monitor) handleMessage(msg Message, addr *net.UnixAddr) {
	m.log.Printf("[DEBUG] Handle message from %s: %s\n",
		addr,
		spew.Sdump(&msg))

	var (
		err error
		req []string
		cmd request.ID
		res Response
		str string
	)

	if req, err = shlex.Split(msg.Request); err != nil {
		m.log.Printf("[ERROR] Cannot parse Request: %s\nRaw: %s\n",
			err.Error(),
			msg.Request)
		return
	} else if cmd, err = request.Parse(req[0]); err != nil {
		m.log.Printf("[ERROR] Don't understand request %q: %s\n",
			req[0],
			err.Error())
	}

	var db = m.pool.Get()
	defer m.pool.Put(db)

	switch cmd {
	case request.JobSubmit:
		if err = db.JobSubmit(msg.Job); err != nil {
			str = fmt.Sprintf("Failed to submit Job: %s",
				err.Error())
			m.log.Printf("[ERROR] %s\n", str)
			res = m.makeResponse(str)
		} else {
			str = fmt.Sprintf("Job submitted, Job ID is %d",
				msg.Job.ID)
			m.log.Printf("[DEBUG] %s\n", str)
			res = m.makeResponse(str)
		}
	default:
		str = fmt.Sprintf("I don't know how to handle %s", cmd)
		m.log.Printf("[INFO] %s\n", str)
		res = m.makeResponse(str)
	}

	var (
		buf []byte
		cnt int
	)

	if buf, err = json.Marshal(&res); err != nil {
		m.log.Printf("[ERROR] Cannot serialize Response to %s: %s\n",
			addr,
			err.Error())
	} else if cnt, err = m.ctl.WriteToUnix(buf, addr); err != nil {
		m.log.Printf("[ERROR] Failed to send Response to %s: %s\n",
			addr,
			err.Error())
	} else if cnt != len(buf) {
		// In this day and age, this shouldn't happen, now, should it?
		m.log.Printf("[ERROR] Unexpected number of bytes sent in response: %d (expected %d)\n",
			cnt,
			len(buf))
	}
} // func (m *Monitor) handleMessage(msg *Message, addr *net.UnixAddr)

func (m *Monitor) makeResponse(status string) Response {
	return Response{
		Timestamp: time.Now(),
		Sequence:  m.seqCnt.Add(1),
		Status:    status,
	}
} // func (m *Monitor) makeResponse(status string) Response

func (m *Monitor) jobLoop() {
	for m.active.Load() {
		m.jobStep()
	}
} // func (m *Monitor) jobLoop()

// I really should try to break up this method in two, so I can return the
// database connection to the pool while the job is running.

func (m *Monitor) jobStep() {
	var (
		err              error
		db               *database.Database
		jobs             []*job.Job
		j                *job.Job
		outpath, errpath string
		outbase, errbase string
	)

	db = m.pool.Get()
	defer func() {
		if db != nil {
			m.pool.Put(db)
		}
	}()

	if jobs, err = db.JobGetPending(1); err != nil {
		m.log.Printf("[ERROR] Cannot query pending Jobs: %s\n",
			err.Error())
		return
	} else if len(jobs) == 0 {
		m.log.Println("[TRACE] Database returned 0 pending jobs.")
	}

	j = jobs[0]

	// generate file names for spooling
	outbase = fmt.Sprintf("jobq.%d.out", j.ID)
	errbase = fmt.Sprintf("jobq.%d.err", j.ID)

	outpath = filepath.Join(common.SpoolDir, outbase)
	errpath = filepath.Join(common.SpoolDir, errbase)

	if err = db.JobStart(j); err != nil {
		m.log.Printf("[ERROR] Cannot mark Job %d as started in database: %s\n",
			j.ID,
			err.Error())
		return
	} else if err = j.Start(outpath, errpath); err != nil {
		m.log.Printf("[ERROR] Failed to start job %d: %s\n",
			j.ID,
			err.Error())
		return // Really? Just bail? No! FIXME
	}

	m.pool.Put(db)
	db = nil

	// Wait for iiiiit. Literally.
	if err = j.Wait(); err != nil {
		m.log.Printf("[ERROR] Job %d failed: %s\n",
			j.ID,
			err.Error())
	}

	db = m.pool.Get()

	if err = db.JobFinish(j); err != nil {
		m.log.Printf("[ERROR] Failed to mark Job %d as finished: %s\n",
			j.ID,
			err.Error())
	}
} // func (m *Monitor) jobStep() error
