// /home/krylon/go/src/github.com/blicero/jobq/monitor/01_monitor_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-16 14:29:24 krylon>

package monitor

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/blicero/jobq/job"
	"github.com/blicero/jobq/monitor/request"
)

const netname = "unixpacket"

var mon *Monitor

func TestMonCreate(t *testing.T) {
	const name = "TestMonitor"

	var (
		err  error
		path = fmt.Sprintf("/tmp/jobq.%s.%s.%d",
			os.Getenv("USER"),
			name,
			os.Getpid())
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
		err         error
		raddr       net.UnixAddr
		conn        *net.UnixConn
		directories = []string{
			"/etc",
			"/usr/lib",
			"/usr/include",
		}
	)

	raddr = net.UnixAddr{
		Net:  netname,
		Name: socketPath,
	}

	// laddr = net.UnixAddr{
	// 	Net:  netname,
	// 	Name: "client",
	// }

	if conn, err = net.DialUnix(netname, nil, &raddr); err != nil {
		t.Fatalf("Error connecting to Monitor %s: %s",
			socketPath,
			err.Error())
	}

	defer conn.Close() // nolint: errcheck

	// t.Logf("LOCAL ADDRESS = %s", laddr)

	for _, d := range directories {
		var (
			buf []byte
			j   *job.Job
			msg Message
			opt = job.Options{
				Directory: d,
				Compress:  "yes",
			}
		)

		if j, err = job.New(opt, "/bin/ls", "-lh"); err != nil {
			t.Errorf("Failed to create Job: %s", err.Error())
			continue
		}

		msg = MakeMsg(request.JobSubmit.String(), j)

		if buf, err = json.Marshal(&msg); err != nil {
			t.Fatalf("Cannot serialize Job: %s",
				err.Error())
		} else if _, err = conn.Write(buf); err != nil {
			t.Errorf("Cannot send JSON payload to Monitor: %s",
				err.Error())
		}
	}

	time.Sleep(time.Second * 30)
} // func TestMonSubmit(t *testing.T)
