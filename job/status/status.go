// /home/krylon/go/src/github.com/blicero/jobq/job/status/status.go
// -*- mode: go; coding: utf-8; -*-
// Created on 31. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-31 19:41:19 krylon>

// Package status provides symbolic constants to describe the life cycle
// of a Job.
package status

//go:generate stringer -type=Status

// Status represents the status of a Job.
type Status uint8

const (
	Created Status = iota
	Enqueued
	Started
	Finished
)
