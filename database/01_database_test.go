// /home/krylon/go/src/github.com/blicero/jobq/database/01_database_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-08-01 21:32:11 krylon>

package database

import (
	"testing"

	"github.com/blicero/jobq/common"
	"github.com/blicero/jobq/job"
)

var db *Database

func TestDBOpen(t *testing.T) {
	var err error

	if db, err = Open(common.DbPath); err != nil {
		db = nil
		t.Fatalf("Cannot open database at %s: %s",
			common.DbPath,
			err.Error())
	}
} // func TestDBOpen(t *testing.T)

func TestParseQueries(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	for qid := range qDB {
		var err error
		if _, err = db.getQuery(qid); err != nil {
			t.Errorf("Cannot prepare query %s: %s",
				qid,
				err.Error())
		}
	}
} // func TestParseQueries(t *testing.T)

var tj *job.Job

func TestJobSubmit(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	var (
		err error
		j   *job.Job
		opt = job.Options{
			Directory: "/etc",
		}
	)

	if j, err = job.New(opt, "/bin/ls", "-lh"); err != nil {
		t.Fatalf("Cannot create new Job: %s",
			err.Error())
	} else if err = db.JobSubmit(j); err != nil {
		t.Fatalf("Error submitting Job: %s",
			err.Error())
	} else if j.ID == 0 {
		t.Fatal("Job ID after submission must not be 0")
	}

	tj = j

	var j2 *job.Job

	if j2, err = db.JobGetByID(j.ID); err != nil {
		t.Fatalf("Failed to fetch Job from Database: %s",
			err.Error())
	} else if j2 == nil {
		t.Fatalf("Looking for Job #%d should not return nil",
			j.ID)
	}
} // func TestJobSubmit(t *testing.T)

func TestJobGetPending(t *testing.T) {
	if db == nil || tj == nil {
		t.SkipNow()
	}

	var (
		err  error
		jobs []job.Job
	)

	if jobs, err = db.JobGetPending(-1); err != nil {
		t.Fatalf("Failed to get list of pending Jobs: %s",
			err.Error())
	} else if len(jobs) != 1 {
		t.Fatalf("Unexpected number of Jobs pending: %d (expected 1)",
			len(jobs))
	}
} // func TestJobGetPending(t *testing.T)
