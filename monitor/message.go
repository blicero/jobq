// /home/krylon/go/src/github.com/blicero/jobq/monitor/message.go
// -*- mode: go; coding: utf-8; -*-
// Created on 08. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-10 11:15:05 krylon>

package monitor

import (
	"time"

	"github.com/blicero/jobq/job"
)

// Message is data format for communication between client and server.
type Message struct {
	Timestamp time.Time
	Job       *job.Job
	Request   string
}

// MakeMsg returns a new message.
func MakeMsg(req string, j *job.Job) Message {
	msg := Message{
		Timestamp: time.Now(),
		Job:       j,
		Request:   req,
	}
	return msg
} // func MakeMsg(req string, j *job.Job)
