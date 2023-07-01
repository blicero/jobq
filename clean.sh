#!/bin/sh
# Time-stamp: <2023-07-01 19:43:27 krylon>

cd $GOPATH/src/github.com/blicero/jobq/

rm -vf bak.jobq jobq dbg.build.log \
    && du -sh . \
    && git fsck --full \
    && git reflog expire --expire=now \
    && git gc --aggressive --prune=now \
    && du -sh .


