// /home/krylon/go/src/github.com/blicero/jobq/job/job.go
// -*- mode: go; coding: utf-8; -*-
// Created on 18. 06. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-07 16:11:30 krylon>

// Package job provides the Job type.
package job

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Error is an error type to represent errors related to the lifecycle
// of a Job.
type Error struct {
	Message  string
	Previous error
}

func makeJobError(m string, previous error) *Error {
	return &Error{
		Message:  m,
		Previous: previous,
	}
} // func makeJobError(m string, previous error) *JobError

func (je *Error) Unwrap() error {
	return je.Previous
} // func (je *JobError) Unwrap()

func (je *Error) Error() string {
	if je.Message == "" {
		return je.Previous.Error()
	}

	return fmt.Sprintf("%s: %s",
		je.Message,
		je.Previous.Error())
} // func (je *JobError) Error() string

// Various predefine error values:
//
// ErrJobStarted indicates that a Job could not be started because it had started already.
//
// ErrJobNotStarted indicates that an operation failed because the Job has
// not been started yet.
//
// ErrInvalidOption indicates that the Options used for the Job contain an invalid value.
var (
	ErrJobStarted    = errors.New("Job has been started already")
	ErrJobNotStarted = errors.New("Job has not been started")
	ErrInvalidOption = errors.New("Invalid Option")
)

// Options for the Job
type Options struct {
	MaxDuration time.Duration
	Directory   string
	Compress    string
	Nice        int
}

// Job is a batch job, submitted for execution.
// ID is an integer value that is used to uniquely identify Job instances
//
// Options is of type Options, see there for further reference.
//
// TimeSubmitted is the time the Job was submitted to the queue. To be filled
// in by the Job queue or scheduler.
//
// TimeStarted and TimeEnded are the times at which the Job was started and
// ended, to be filled in by the scheduler or monitor.
//
// ExitCode is the exit code given by the operating system.
//
// Cmd is the array of arguments, the first element is the command itself,
// followed by parameters/arguments.
//
// SpoolOut and SpoolErr are the names of the files where the output of the
// Job is stored, again to be filled in by the scheduler.
//
// proc (private) is a handle to process while it is running.
type Job struct {
	Options
	ID            int64
	TimeSubmitted time.Time
	TimeStarted   time.Time
	TimeEnded     time.Time
	ExitCode      int
	Cmd           []string
	SpoolOut      string
	SpoolErr      string
	proc          *exec.Cmd
}

// New creates a new Job instance with the given options and command line.
//
// Currently, the error value returned is always nil, but in the future, this
// might change.
func New(options Options, cmd ...string) (*Job, error) {
	var (
		j = &Job{
			Options:  options,
			Cmd:      cmd,
			ExitCode: -1,
		}
	)

	return j, nil
} // func New(maxdur time.Duration, cmd ...string) (*Job, error)

// CmdString returns the Job's command line as a single string.
func (j *Job) CmdString() string {
	//return strings.Join(j.Cmd, " ")
	var (
		err error
		buf []byte
	)

	if buf, err = json.Marshal(j.Cmd); err != nil {
		panic(err)
	}

	return string(buf)
} // func (j *Job) CmdString() string

// Start attempts to prepare everything needed for the Job's execution and
// then start it.
func (j *Job) Start(outpath, errpath string) error {
	var (
		err        error
		outh, errh *os.File
		outc, errc io.Writer
	)

	if j.proc != nil {
		return ErrJobStarted
	}

	j.SpoolOut = outpath
	j.SpoolErr = errpath

	if outh, err = os.Create(outpath); err != nil {
		return makeJobError(
			fmt.Sprintf("Error opening spool file for stdout %q", outpath),
			err)
	} else if errh, err = os.Create(errpath); err != nil {
		return makeJobError(
			fmt.Sprintf("Error opening spool file for stderr %q", outpath),
			err)
	}

	switch strings.ToLower(j.Options.Compress) {
	case "", "no", "false":
		outc = outh
		errc = errh
	case "gzip", "yes", "true":
		outc = gzip.NewWriter(outh)
		errc = gzip.NewWriter(errh)
	default:
		return makeJobError(
			fmt.Sprintf("Invalid compression type %q", j.Options.Compress),
			ErrInvalidOption)
	}

	j.proc = exec.Command(j.Cmd[0], j.Cmd[1:]...)

	j.proc.Stdout = outc
	j.proc.Stderr = errc
	j.proc.Dir = j.Directory

	// more stuff

	if err = j.proc.Start(); err != nil {
		return makeJobError(
			fmt.Sprintf("Error starting command %s", j.Cmd[0]),
			err)
	}

	j.TimeStarted = time.Now()

	return nil
} // func (j *Job) Start() error

// Wait waits for a started Job to finish and does the post-processing.
func (j *Job) Wait() error {
	var (
		err error
	)

	if j.proc == nil || j.proc.Process == nil {
		return ErrJobNotStarted
	} else if err = j.proc.Wait(); err != nil {
		// Deal with it! FIXME
		fmt.Fprintf(
			os.Stderr,
			"Error running Job %d (%s): %s\n",
			j.ID,
			j.Cmd[0],
			err.Error())
	}

	j.TimeEnded = time.Now()

	return err
} // func (j *Job) Wait() error
