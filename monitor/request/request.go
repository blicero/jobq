// /home/krylon/go/src/github.com/blicero/jobq/monitor/request/request.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-08-01 22:39:28 krylon>

package request

import "fmt"

//go:generate stringer -type=ID

// ID represents a type of request sent to the Monitor
type ID uint8

// These constants represent the kinds of requests the Monitor handles
const (
	Invalid ID = iota
	JobSubmit
	JobCancel
	JobClear
	JobQueryStatus
	QueueQueryStatus
	MonitorStop
	MonitorRestart // ???
)

// Parse attempts to convert a string to an ID value.
func Parse(s string) (ID, error) {
	var id ID
	switch s {
	case "JobSubmit":
		id = JobSubmit
	case "JobCancel":
		id = JobCancel
	case "JobClear":
		id = JobClear
	case "JobQueryStatus":
		id = JobQueryStatus
	case "QueueQueryStatus":
		id = QueueQueryStatus
	case "MonitorStop":
		id = MonitorStop
	case "MonitorRestart":
		id = MonitorRestart
	default:
		return Invalid, fmt.Errorf("Invalid Request type %q", s)
	}

	return id, nil
} // func Parse(s string) (ID, error)
