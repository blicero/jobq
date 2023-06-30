// /home/krylon/go/src/github.com/blicero/jobq/queue/02_queue_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 30. 06. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-06-30 20:04:54 krylon>

package queue

import (
	"strconv"
	"testing"
	"time"

	"github.com/blicero/jobq/common"
	"github.com/blicero/jobq/job"
)

const testJobCnt = 10

var jq *Queue

func TestQueueCreate(t *testing.T) {
	var err error

	if jq, err = New(); err != nil {
		jq = nil
		t.Fatalf("Failed to create Queue: %s",
			err.Error())
	}

	jq.Start()
} // func TestQueueCreate(t *testing.T)

func TestSubmit(t *testing.T) {
	if jq == nil || !jq.Active() {
		t.SkipNow()
	}

	const (
		command = "/bin/sleep"
		delay   = 5
	)

	var (
		err  error
		j    *job.Job
		opts = job.Options{
			Directory: common.SpoolDir,
		}
	)

	for i := 0; i < testJobCnt; i++ {
		if j, err = job.New(opts, command, strconv.Itoa(delay)); err != nil {
			t.Fatalf("Failed to create job: %s", err.Error())
		} else if err = jq.Submit(j); err != nil {
			t.Errorf("Error submitting job %02d: %s",
				i+1, err.Error())
		}
	}

	time.Sleep(time.Second*delay*testJobCnt + 1)

	if l := jq.Length(); l != 0 {
		t.Errorf("Job queue should be empty, but there are %d jobs pending",
			l)
	}
} // func TestSubmit(t *testing.T)
