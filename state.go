package main

import (
  "os"
  "fmt"
  "time"
  "encoding/json"
  "path/filepath"
)

// State holds a map for matching paths to Logfile structs
// and the File this object was loaded from. The File will
// be used later to save the state.
type State struct {
  TS        time.Time
  File      string
  Logfiles  map[string]Logfile
}

// Logfile contains the FileState, which tries to identify
// a file, and the positive matches.
type Logfile struct {
	File		*FileState
  Matches map[string]Match
}

// Match just maps ignore (or inverse) match patterns to the
// InvMatch structs holding the Offsets.
type Match struct {
  InvMatches  map[string]InvMatch
}

// InvMatch olds the actual offset.
type InvMatch struct {
  Offset  int64
  Time    time.Time
}

// GetOffset looks up the offset for a given triple of (logfile, pattern, ignore pattern).
// If there is not state matching this triple it always returns 0.
func (s *State) GetOffset( logfile string, pattern string, invPat string ) int64 {
  // first see if we've already seen this filename
  l, found := s.Logfiles[logfile]
  if !found {
    return 0
  }
  // if we've already seen this filename we compare it's deviceid/inode in order
  // to make sure that we count right
	if !l.File.SameFile(logfile) {
		return 0
	}
  // ok, it's still the same file. now let's see if we've already
  // seen this RegExp
  match, found := l.Matches[pattern]
  if found {
    // now see if we also have seen the invPattern (may be an empty string)
    invMatch, found := match.InvMatches[invPat]
    if found {
      return invMatch.Offset
    } else {
      return 0
    }
  }
  return 0
}

// SetOffset writes the offset for a given triple of (logfile, pattern, ignore pattern)
// to the state struct. It will also record the file metadata of the logfile for
// comparison after loading the state again.
func (s *State) SetOffset( logfile string, pattern string, invPat string, offset int64 ) {
  fi, err := NewFileState( logfile )
  if err != nil {
    fmt.Println("could not access metadata of file %s: %s\n", logfile, err)
  }

  l, found := s.Logfiles[logfile]
  // new file
  if !found {
    matches := make(map[string]Match)
    l = Logfile{
      File: fi,
      Matches: matches,
    }
  }

  // file changed (differnt size and/or inode)
  if !l.File.SameState(fi) {
    matches := make(map[string]Match)
    l.Matches = matches
    l.File = fi
  }

  // pattern known?
  match, matchFound := l.Matches[pattern]
  if !matchFound {
    match = Match{
      InvMatches: make(map[string]InvMatch),
    }
  }

  // don't care if we've already seen this inv pattern,
  // since we're going to overwrite it anyway
  match.InvMatches[invPat] = InvMatch{
    Offset: offset,
    Time: time.Now(),
  }

  l.Matches[pattern] = match
  s.Logfiles[logfile] = l
  return
}

func validStateFile(filename string) bool {
  // too short to be a valid filename
  if len(filename) < 1 {
    return false
  }
  // if the file exists: OK
  if _, err := os.Stat(filename); err == nil {
    return true
  }
  // if not: see if we can write to the directory
  dir := filepath.Dir(filename)
  if f, err := os.Stat(dir); err != nil && f.IsDir() {
    return true
  }
  // looks invalid
  return false
}

func getStateFile() string {
  if validStateFile( userStateFile ) {
    return userStateFile
  }
  // try to use the homedir as our statefile location
  stateDir := os.Getenv("HOME")
  // if that does not exist (possible for daemon users)
  // fall back to the current dir
  if _, err := os.Stat(stateDir); os.IsNotExist(err) {
    stateDir = "."
  }

  return stateDir + "/.check-log.state"
}

func NewState() *State {
  return &State{
    TS: time.Now(),
    Logfiles: make(map[string]Logfile),
  }
}

func LoadState( filename string ) *State {
  var stateFile string
  if len(filename) > 0 {
    stateFile = filename
  } else {
    stateFile = getStateFile()
  }
  state := NewState()

  if _, err := os.Stat(stateFile); os.IsNotExist(err) {
    state.File = stateFile
    return state
  }
  file, err := os.OpenFile( stateFile, os.O_RDONLY, 0644 )
  if err != nil {
    state.File = stateFile
    return state
  }
  dec := json.NewDecoder(file)
  err = dec.Decode(&state)
  if err != nil {
    fmt.Println("decode error:", err)
  }
  state.File = stateFile
  return state
}

func (s *State) Save() {
  stateFile := s.File

  file, err := os.OpenFile( stateFile, os.O_WRONLY|os.O_CREATE, 0644 )
  if err != nil {
    fmt.Printf("could not open statefile %s for writing: %s\n", s.File, err)
    return
  }
  defer file.Close()

  enc := json.NewEncoder(file)
  s.TS = time.Now()
  err = enc.Encode(s)
  if err != nil {
    fmt.Println("encode error:", err)
    return
  }
}

