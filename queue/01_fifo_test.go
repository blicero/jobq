// /home/krylon/go/src/github.com/blicero/jobq/queue/01_fifo_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 06. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-06-20 00:16:07 krylon>

package queue

import (
	"testing"

	"github.com/blicero/jobq/job"
)

var tq fifo // tq = "test queue"

func TestFIFOCreate(t *testing.T) {
	fifoInit(&tq)

	if l := tq.length(); l != 0 {
		t.Fatalf("Freshly initialized queue has length %d, should be 0", l)
	}
} // func TestFIFOCreate(t *testing.T)

func TestFIFOEnq(t *testing.T) {
	const cnt = 5
	var opts = job.Options{
		Directory: "/tmp",
	}

	for i := 0; i < cnt; i++ {
		var (
			err error
			j   *job.Job
		)

		if j, err = job.New(opts, "/bin/ls", "/tmp", "/etc"); err != nil {
			t.Fatalf("Cannot create new job: %s", err.Error())
		}

		tq.enqueue(j)

		if l := tq.length(); l != i+1 {
			t.Errorf("Unexpected queue length after insertion: %d (expected %d)",
				l,
				i+1)
		}
	}
} // func TestFIFOEnq(t *testing.T)
