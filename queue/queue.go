// /home/krylon/go/src/github.com/blicero/jobq/queue/queue.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 06. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-06-19 20:22:18 krylon>

// Package queue implements the queueing of jobs.
package queue

import (
	"fmt"
	"log"
	"path/filepath"
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

	fifoInit(&q.q)

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

// Start activates the queue.
func (q *Queue) Start() {
	go q.loop()
} // func (q *Queue) Start()

// "private"

// The Queue's main loop, this is meant to be run in a separate goroutine.
func (q *Queue) loop() {
	q.active.Store(true)
	defer q.active.Store(false)

	for {
		var (
			err              error
			j                *job.Job
			outpath, errpath string
			outbase, errbase string
		)

		j = q.q.dequeue()

		// generate file names for spooling
		outbase = fmt.Sprintf("jobq.%d.out", j.ID)
		errbase = fmt.Sprintf("jobq.%d.err", j.ID)

		outpath = filepath.Join(common.SpoolDir, outbase)
		errpath = filepath.Join(common.SpoolDir, errbase)

		if err = j.Start(outpath, errpath); err != nil {
			q.log.Printf("[ERROR] Failed to start job %d: %s\n",
				j.ID,
				err.Error())
			continue // Really? Just bail? No! FIXME
		}

		// Wait for iiiiit. Literally.
		if err = j.Wait(); err != nil {
			q.log.Printf("[ERROR] Job %d failed: %s\n",
				j.ID,
				err.Error())
		}
	}
} // func (q *Queue) loop()
