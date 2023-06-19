// /home/krylon/go/src/github.com/blicero/jobq/queue/queue.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 06. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-06-19 18:00:53 krylon>

// Package queue implements the queueing of jobs.
package queue

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/blicero/jobq/common"
	"github.com/blicero/jobq/job"
	"github.com/blicero/jobq/logdomain"
)

// Queue is the job queue.
type Queue struct {
	q      fifo
	log    *log.Logger
	active atomic.Bool // nolint: unused
}

// New creates a new Job Queue.
func New() (*Queue, error) {
	var (
		err error
		q   = new(Queue)
	)

	if q.log, err = common.GetLogger(logdomain.Queue); err != nil {
		return nil, err
	}

	return q, nil
} // func New() (*Queue, error)

// Length returns the number of pending jobs.
func (q *Queue) Length() int {
	return q.q.length()
} // func (q *Queue) Length() int

// Submit adds a new Job to the Queue.
func (q *Queue) Submit(j *job.Job) error {
	j.TimeSubmitted = time.Now()

	q.q.enqueue(j)

	return nil
} // func (q *Queue) Submit(j *job.Job) error

// "private"

func (q *Queue) loop() {
	q.active.Store(true)
	defer q.active.Store(false)

} // func (q *Queue) loop()
