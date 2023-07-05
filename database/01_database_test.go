// /home/krylon/go/src/github.com/blicero/jobq/database/01_database_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-05 20:03:52 krylon>

package database

import (
	"testing"

	"github.com/blicero/jobq/common"
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
