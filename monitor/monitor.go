// /home/krylon/go/src/github.com/blicero/jobq/monitor/monitor.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-08-02 18:12:10 krylon>

// Package monitor is the nexus of the batch system.
package monitor

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"path/filepath"
	"strings"
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
	name      string
	path      string
	log       *log.Logger
	pool      *database.Pool
	active    atomic.Bool
	slots     int
	ctl       *net.UnixListener
	seqCnt    atomic.Int64
	jobq      chan int
	jobTicker *time.Ticker
}

// Create creates and returns a new Monitor.
func Create(name, sock string, slots int) (*Monitor, error) {
	var (
		err error
		m   = &Monitor{
			name:      name,
			path:      sock,
			slots:     slots,
			jobq:      make(chan int),
			jobTicker: time.NewTicker(time.Minute * 5),
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
	} else if m.ctl, err = net.ListenUnix("unixpacket", &addr); err != nil {
		m.log.Printf("[ERROR] Cannot open control socket %s: %s\n",
			sock,
			err.Error())
		return nil, err
	}

	return m, nil
} // func Create(name, sock string, slots int) (*Monitor, error)

func (m *Monitor) jobTick() {
	var t = time.NewTimer(time.Second * 10)
	select {
	case m.jobq <- 1:
		if !t.Stop() {
			<-t.C
		}
	case <-t.C:
		m.log.Printf("[ERROR] Timeout waiting to send job signal\n")
	}
} // func (m *Monitor) jobTick()

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
	for m.active.Load() {
		var (
			err  error
			conn *net.UnixConn
		)

		// I should probably set a timeout for reading, so I can check
		// the active flag periodically?
		if conn, err = m.ctl.AcceptUnix(); err != nil {
			m.log.Printf("[ERROR] Error accepting new connection: %s\n",
				err.Error())
		}

		go m.handleClient(conn)
	}
} // func (m *Monitor) ctlLoop()

func (m *Monitor) handleClient(client *net.UnixConn) {
	const maxErr = 5
	var (
		err         error
		msg         Message
		cnt, errcnt int
		buffer      = make([]byte, 65536)
	)

	defer client.Close() // nolint: errcheck

	for m.active.Load() && errcnt < maxErr {
		if cnt, err = client.Read(buffer); err != nil {
			if err == io.EOF {
				m.log.Println("[INFO] Client closed connection.")
				break
			}
			m.log.Printf("[ERROR] Failed to read from Client: %s\n",
				err.Error())
			errcnt++
			continue
		} else if err = json.Unmarshal(buffer[:cnt], &msg); err != nil {
			m.log.Printf("[ERROR] Failed to decode message: %s\nRaw: %s\n\n",
				err.Error(),
				string(buffer[:cnt]))
			errcnt++
			continue
		} else if err = m.handleMessage(msg, client); err != nil {
			m.log.Printf("[ERROR] Error handling message from %s: %s\n",
				client.RemoteAddr(),
				err.Error())
			errcnt++
			continue
		}
	}
} // func (m *Monitor) handleClient(client net.Conn)

func (m *Monitor) handleMessage(msg Message, conn *net.UnixConn) error {
	m.log.Printf("[DEBUG] Handle message: %s\n",
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
		return err
	} else if cmd, err = request.Parse(req[0]); err != nil {
		m.log.Printf("[ERROR] Don't understand request %q: %s\n",
			req[0],
			err.Error())
		return err
	}

	var db = m.pool.Get()
	defer m.pool.Put(db)

	switch cmd {
	case request.JobSubmit:
		msg.Job.TimeSubmitted = time.Now()
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
			go m.jobTick()
		}
	case request.JobCancel:
		var response = "Job cancellation is not implemented, yet."
		res = m.makeResponse(response)
		// FIXME: I already wrote most of the code, but I commented it
		// out and moved it to the end of the file for readibility.
	case request.JobClear:
		// remove all finished jobs the database.
	case request.QueueQueryStatus:
		var jobs []job.Job
		if jobs, err = db.JobGetAll(); err != nil {
			str = fmt.Sprintf("Failed to query all Jobs: %s",
				err.Error())
			m.log.Printf("[ERROR] %s\n", str)
			res = m.makeResponse(str)
		} else {
			res = m.makeResponse("OK")
			res.Jobs = jobs
		}
	default:
		str = fmt.Sprintf("I don't know how to handle %s", cmd)
		m.log.Printf("[INFO] %s\n", str)
		res = m.makeResponse(str)
	}

	var (
		buf  []byte
		cnt  int
		addr = conn.RemoteAddr()
	)

	if buf, err = json.Marshal(&res); err != nil {
		m.log.Printf("[ERROR] Cannot serialize Response to %s: %s\n",
			addr,
			err.Error())
		return err
	} else if cnt, err = conn.Write(buf); err != nil {
		m.log.Printf("[ERROR] Failed to send Response to %s: %s\n",
			addr,
			err.Error())
		return err
	} else if cnt != len(buf) {
		// In this day and age, this shouldn't happen, now, should it?
		m.log.Printf("[ERROR] Unexpected number of bytes sent in response: %d (expected %d)\n",
			cnt,
			len(buf))
		return err
	}

	return nil
} // func (m *Monitor) handleMessage(msg Message, conn *net.UnixConn) error

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
		jobs             []job.Job
		j                job.Job
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
		select {
		case <-m.jobTicker.C:
			return
		case <-m.jobq:
			return
		}
	}

	j = jobs[0]

	m.log.Printf("[DEBUG] Starting Job %d, submitted %s ago (%q)\n",
		j.ID,
		time.Since(j.TimeSubmitted),
		strings.Join(j.Cmd, " "))

	// generate file names for spooling
	outbase = fmt.Sprintf("jobq.%d.out", j.ID)
	errbase = fmt.Sprintf("jobq.%d.err", j.ID)

	outpath = filepath.Join(common.SpoolDir, outbase)
	errpath = filepath.Join(common.SpoolDir, errbase)

	if err = j.Start(outpath, errpath); err != nil {
		m.log.Printf("[ERROR] Failed to start job %d: %s\n",
			j.ID,
			err.Error())
		return // Really? Just bail? No! FIXME
	} else if err = db.JobStart(&j); err != nil {
		m.log.Printf("[ERROR] Cannot mark Job %d as started in database: %s\n",
			j.ID,
			err.Error())
		return
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

	if err = db.JobFinish(&j); err != nil {
		m.log.Printf("[ERROR] Failed to mark Job %d as finished: %s\n",
			j.ID,
			err.Error())
	}
} // func (m *Monitor) jobStep() error

//////////////////////////////////////////////////////////////////
// HANDLING JOB CANCELLATION /////////////////////////////////////
//////////////////////////////////////////////////////////////////
// var (
// 	j   *job.Job
// 	jid int64
// )

// if jid, err = strconv.ParseInt(req[1], 10, 64); err != nil {
// 	str = fmt.Sprintf("Cannot parse Job ID %q: %s",
// 		req[1],
// 		err.Error())
// 	m.log.Printf("[ERROR] %s\n", str)
// 	res = m.makeResponse(str)
// } else if j, err = db.JobGetByID(jid); err != nil {
// 	str = fmt.Sprintf("Error looking up Job %d: %s",
// 		jid,
// 		err.Error())
// 	m.log.Printf("[ERROR] %s\n", str)
// 	res = m.makeResponse(str)
// } else if j == nil {
// 	str = fmt.Sprintf("Did not find Job %d in database",
// 		jid)
// 	m.log.Printf("[ERROR] %s\n", str)
// 	res = m.makeResponse(str)
// }

// // We can safely ignore status.Created, because a Job that is not
// // "in the system" yet can simply be discarded.
// switch j.Status() {
// case status.Enqueued:
// 	if err = db.JobDelete(j); err != nil {
// 		m.log.Printf("[ERROR] Cannot delete Job %d: %s\n",
// 			j.ID,
// 			err.Error())
// 	}
// 	// delete
// case status.Started:
// 	// kill process, delete spool files, delete job
// case status.Finished:
// 	// delete spool files, delete job
// }
