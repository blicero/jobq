// /home/krylon/go/src/github.com/blicero/jobq/job/01_job_create_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 18. 06. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-06-18 22:30:56 krylon>

package job

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/blicero/jobq/common"
)

type testCmd struct {
	cmd         []string
	expectError bool
	opts        Options
}

func TestJobCreate(t *testing.T) {
	var testJobs = []testCmd{
		testCmd{
			cmd:  []string{"/bin/ls", "/tmp"},
			opts: Options{},
		},
		testCmd{
			cmd: []string{"find", os.Getenv("HOME"), "-type", "d"},
			opts: Options{
				Compress: "yes",
			},
		},
		testCmd{
			cmd:         []string{"/bin/rm", "/no.such.file"},
			expectError: true,
		},
	}

	for idx, tj := range testJobs {
		var (
			err error
			j   *Job
		)

		if j, err = New(tj.opts, tj.cmd...); err != nil {
			t.Errorf("Error creating Job %d: %s",
				idx,
				err.Error())
		}

		var (
			outpath, errpath string
		)

		outpath = filepath.Join(common.BaseDir, fmt.Sprintf("out.%d", j.ID))
		errpath = filepath.Join(common.BaseDir, fmt.Sprintf("err.%d", j.ID))

		if err = j.Start(outpath, errpath); err != nil {
			t.Errorf("Failed to start Job %d: %s",
				j.ID,
				err.Error())
		} else if err = j.Wait(); err != nil && !tj.expectError {
			t.Errorf("Running Job %d returned an error: %s",
				j.ID,
				err.Error())
		} else if err == nil && tj.expectError {
			t.Errorf("Job %d ran without error, but we expected it to fail",
				j.ID)
		}
	}
}
