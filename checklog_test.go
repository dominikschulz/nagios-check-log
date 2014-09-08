package main

import (
  "testing"
  "io/ioutil"
  "os"
)

func TestParser(t *testing.T) {
  var logfiles = make([]string, 1)

  // create tempdir for fs based tests
  tempdir, _ := ioutil.TempDir(os.TempDir(), "checklog-tests-")
  logfile := tempdir + "/log"
  _ = ioutil.WriteFile( logfile, []byte("foo\nfoobar\nfoo"), 0644)

  // test checkLogs
  logfiles[0] = logfile
  found := checkLogs( logfiles, "foo", "foobar" )
  if found != 2 {
    t.Errorf("expected exactly two occurences of foo")
  }

  // test globFiles
  _ = ioutil.WriteFile( logfile + ".1.log", []byte("foo\nfoobar\nfoo"), 0644)
  _ = ioutil.WriteFile( logfile + ".2.log", []byte("foo\nfoobar\nfoo"), 0644)
  _ = ioutil.WriteFile( logfile + ".3.log", []byte("foo\nfoobar\nfoo"), 0644)
  _ = ioutil.WriteFile( logfile + ".4.log", []byte("foo\nfoobar\nfoo"), 0644)
  globPat := logfile + ".*.log"
  logfiles = globFiles( globPat )
  if len(logfiles) != 4 {
    t.Errorf("expected exactly 4 logfiles matching %s not %d\n", globPat, len(logfiles) )
  }

  // remove tempdir
  _ = os.RemoveAll(tempdir)
}

