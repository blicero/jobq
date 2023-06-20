// /home/krylon/go/src/github.com/blicero/jobq/queue/01_fifo_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 06. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-06-20 06:38:25 krylon>

package queue

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/blicero/jobq/job"
)

const (
	testCnt  = 5
	extraCnt = 2
)

var tq fifo // tq = "test queue"

func TestFIFOCreate(t *testing.T) {
	fifoInit(&tq)

	if l := tq.length(); l != 0 {
		t.Fatalf("Freshly initialized queue has length %d, should be 0", l)
	}
} // func TestFIFOCreate(t *testing.T)

func TestFIFOEnq(t *testing.T) {
	var opts = job.Options{
		Directory: "/tmp",
	}

	for i := 0; i < testCnt; i++ {
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

func TestFIFODeqNB(t *testing.T) {
	for i := testCnt - 1; i >= 0; i-- {
		if j := tq.dequeueNB(); j == nil {
			t.Fatal("dequeueNB returned nil while fifo was not empty")
		} else if l := tq.length(); l != i {
			t.Fatalf("Unexpected fifo length: %d (expected %d)",
				l, i)
		}
	}

	if j := tq.dequeueNB(); j != nil {
		t.Errorf("dequeueNB should have return nil (fifo empty), but we got %#v",
			j)
	}
} // func TestFIFODeqNB(t *testing.T)

func TestFIFOBlock(t *testing.T) {
	var opts = job.Options{
		Directory: "/tmp",
	}

	// First, we fill up the fifo again.
	for i := 0; i < testCnt; i++ {
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

	var dcnt atomic.Int32

	go func() {
		var i int32
		for i = 0; i < testCnt+extraCnt; i++ {
			_ = tq.dequeue()
			t.Logf("Removed %d elements from fifo so far", i+1)
			dcnt.Store(i + 1)
		}

		time.Sleep(time.Second)
		dcnt.Store(-testCnt)
	}()

	time.Sleep(time.Millisecond * 200)

	if c := dcnt.Load(); c != testCnt {
		t.Fatalf("Unexpected number of elements removed from queue: %d (expected %d)",
			c,
			testCnt)
	}

	for i := 0; i < extraCnt; i++ {
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

	time.Sleep(time.Millisecond * 50)

	if c := dcnt.Load(); c != testCnt+extraCnt {
		t.Fatalf("Unexpected number of elements removed from queue: %d (expected %d)",
			c,
			testCnt+extraCnt)
	}

	time.Sleep(time.Millisecond * 1200)

	if c := dcnt.Load(); c != -testCnt {
		t.Fatalf("Unexpected value in counter: %d (expected %d)",
			c,
			-testCnt)
	}

	if j := tq.dequeueNB(); j != nil {
		t.Fatalf("dequeueNB on empty fifo should return nil")
	}
} // func TestFIFOBlock(t *testing.T)
