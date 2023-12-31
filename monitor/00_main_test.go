// /home/krylon/go/src/github.com/blicero/jobq/database/00_main_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-17 21:29:35 krylon>

package monitor

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/blicero/jobq/common"
)

var socketPath string

func TestMain(m *testing.M) {
	var (
		err     error
		result  int
		baseDir = time.Now().Format("/tmp/jobq_monitor_test_20060102_150405")
	)

	defer func() {
		if socketPath != "" {
			os.Remove(socketPath) // nolint: errcheck
		}
	}()

	if err = common.SetBaseDir(baseDir); err != nil {
		fmt.Printf("Cannot set base directory to %s: %s\n",
			baseDir,
			err.Error())
		os.Exit(1)
	} else if result = m.Run(); result == 0 {
		// If any test failed, we keep the test directory (and the
		// database inside it) around, so we can manually inspect it
		// if needed.
		// If all tests pass, OTOH, we can safely remove the directory.
		// fmt.Printf("Removing BaseDir %s\n",
		// 	baseDir)
		// _ = os.RemoveAll(baseDir)
	} else {
		fmt.Printf(">>> TEST DIRECTORY: %s\n", baseDir)
	}

	os.Exit(result)
} // func TestMain(m *testing.M)
