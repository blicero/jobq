// /home/krylon/go/src/github.com/blicero/jobq/job/job.go
// -*- mode: go; coding: utf-8; -*-
// Created on 18. 06. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-06-18 16:06:05 krylon>

// Package job provides the Job type.
package job

import (
	"os/exec"
	"time"
)

// Job is a batch job, submitted for execution.
type Job struct {
	ID            int64
	MaxDuration   time.Duration
	TimeSubmitted time.Time
	TimeStarted   time.Time
	TimeEnded     time.Time
	ExitCode      int
	Cmd           []string
	Compress      bool
	SpoolOut      string
	SpoolErr      string
	Proc          *exec.Cmd
}
