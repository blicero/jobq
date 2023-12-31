// /home/krylon/go/src/github.com/blicero/jobq/monitor/message.go
// -*- mode: go; coding: utf-8; -*-
// Created on 08. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-08-01 22:09:04 krylon>

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

// Response is the basic response the Monitor sends after handling a Message.
type Response struct {
	Timestamp time.Time
	Sequence  int64
	Status    string
	Jobs      []job.Job
}
