// /home/krylon/go/src/github.com/blicero/jobq/queue/queue.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 06. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-06-19 14:42:00 krylon>

// Package queue implements the queueing of jobs.
package queue

import (
	"log"
	"sync"

	"github.com/blicero/jobq/common"
	"github.com/blicero/jobq/job"
	"github.com/blicero/jobq/logdomain"
)

// nolint: unused
type qlink struct {
	job  *job.Job
	next *qlink
}

// queue (lowercase-q) implements the queue used internally.
// It's a very simplistic queue, using a linked list, elements are added
// to the back and removed at the front.
// The size of the queue is not restricted. It is synchronized, so access
// by multiple goroutines is safe.
// nolint: unused
type queue struct {
	head *qlink
	tail *qlink
	cnt  int
	lock sync.RWMutex
}

// nolint: unused
func (q *queue) enqueue(j *job.Job) {
	q.lock.Lock()

	q.tail = &qlink{job: j, next: q.tail}
	if q.head == nil {
		q.head = q.tail
	}

	q.cnt++

	q.lock.Unlock()
} // func (q *queue) enqueue(j *job.Job)

// nolint: unused
func (q *queue) dequeue() *job.Job {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.head == nil {
		return nil
	}

	var j = q.head.job
	q.head = q.head.next
	q.cnt--

	return j
} // func (q *queue) dequeue() *job.Job

// Queue is the job queue.
type Queue struct {
	q   *queue // nolint: unused
	log *log.Logger
}

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
