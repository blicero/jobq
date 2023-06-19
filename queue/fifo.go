// /home/krylon/go/src/github.com/blicero/jobq/queue/fifo.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 06. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-06-19 20:20:46 krylon>

package queue

import (
	"sync"

	"github.com/blicero/jobq/job"
)

type fifoLink struct {
	job  *job.Job
	next *fifoLink
}

// fifo (lowercase-q) implements the fifo used internally.
// It's a very simplistic fifo, using a linked list, elements are added
// to the back and removed at the front.
// The size of the fifo is not restricted. It is synchronized, so access
// by multiple goroutines is safe.
type fifo struct {
	head     *fifoLink
	tail     *fifoLink
	cnt      int
	lock     sync.RWMutex
	notempty *sync.Cond
}

func fifoInit(f *fifo) {
	f.notempty = sync.NewCond(&f.lock)
} // func fifoInit(f *fifo)

func (q *fifo) length() int {
	q.lock.RLock()
	var cnt = q.cnt
	q.lock.RUnlock()
	return cnt
} // func (q *queue) length() int

func (q *fifo) enqueue(j *job.Job) {
	q.lock.Lock()

	q.tail = &fifoLink{job: j, next: q.tail}
	if q.head == nil { // i.e. fifo is empty
		q.head = q.tail
		q.notempty.Signal()
	}

	q.cnt++

	q.lock.Unlock()
} // func (q *queue) enqueue(j *job.Job)

// nolint: unused
func (q *fifo) dequeueNB() *job.Job {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.head == nil {
		return nil
	}

	var j = q.head.job
	q.head = q.head.next
	q.cnt--

	return j
} // func (q *queue) dequeueNB() *job.Job

// nolint: unused
func (q *fifo) dequeue() *job.Job {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.cnt == 0 {
		q.notempty.Wait()
	}

	var j = q.head.job
	q.head = q.head.next
	q.cnt--

	return j
} // func (q *fifo) dequeue() *job.Job