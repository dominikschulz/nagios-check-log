package main

import (
  "testing"
  "io/ioutil"
  "os"
)

func TestState(t *testing.T) {
  var err error
  var offset int64
  var readOffset int64
  var pattern string
  var invPat string
  var state *State

  tempdir, _ := ioutil.TempDir(os.TempDir(), "checklog-tests-")
  userStateFile = tempdir + "/state"
  logfile := tempdir + "/log"
  logfile2 := tempdir + "/log2"
  _ = ioutil.WriteFile( logfile, []byte("foo\nfoobar\nfoo"), 0644)

  // Test save and restore
  pattern = "foo"
  invPat  = "bar"
  offset  = 1024

  // make sure the statefile does not (yet) exist
  if _, err := os.Stat(userStateFile); err == nil {
    t.Errorf("UserStateFile at %s already exists before testing!\n", userStateFile)
  }

  // create a new/empty state
  state = LoadState(userStateFile)
  state.SetOffset(logfile, pattern, invPat, offset )
  // save the state to the file
  state.Save()

  // make sure the statefile does exist now
  if _, err := os.Stat(userStateFile); err != nil {
    t.Errorf("UserStateFile at %s does not exist after saving!\n", userStateFile)
  }

  // Now we load the statefile again and see if the offset is the one we've saved above
  state = LoadState(userStateFile)
  readOffset = state.GetOffset(logfile, pattern, invPat)
  if offset != readOffset {
    t.Errorf("Restored offset (%d) differs from saved offset (%d) (%s, %s)\n", readOffset, offset, pattern, invPat)
  }

  // Make sure we get 0 if the logfile changes
  err = os.Remove( logfile )
  if err != nil {
    t.Fatalf("Failed to remove the logfile at %s: %s", logfile, err)
  }
  // if we write the same logfile again we might end up having the same inode
  // this case should also be handled, but this is checked below
  // right now we would like to get a new inode
  err = ioutil.WriteFile( logfile2, []byte("foo\nbar\nfoo\n"), 0644)
  if err != nil {
    t.Fatalf("Failed to write the additional logfile at %s: %s", logfile, err)
  }
  // this file should get a new inode
  err = ioutil.WriteFile( logfile, []byte("foo\nbar\n"), 0644)
  if err != nil {
    t.Fatalf("Failed to write the new logfile at %s: %s", logfile, err)
  }
  // load the state again
  state = LoadState(userStateFile)
  // we should now get an offset 0 since the underlying file should have changed
  offset = state.GetOffset(logfile, pattern, invPat)
  if offset != 0 {
    t.Errorf("Restored offset is %d but should be %d since the file changed", offset, 0)
  }
  // Test empty ignore pattern
  pattern = "foo"
  invPat  = ""
  offset  = 256

  // write a new state to our statefile and save it
  state = LoadState(userStateFile)
  state.SetOffset(logfile, pattern, invPat, offset )
  state.Save()

  // load the state again and make sure the restored offset
  // matches the one we wrote
  state = LoadState(userStateFile)
  readOffset = state.GetOffset(logfile, pattern, invPat)
  if offset != readOffset {
    t.Errorf("Restored offset (%d) differs from saved offset (%d) (%s, %s)\n", readOffset, offset, pattern, invPat)
  }

  // remove tempdir
  _ = os.RemoveAll(tempdir)
}

