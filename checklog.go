package main

import (
  "github.com/gittex/nagiosplugin"
  "fmt"
  "os"
  "io"
  "regexp"
  "bufio"
  "runtime"
  "math"
  "flag"
  "bytes"
  "path/filepath"
)

// Each core should be busy. Create one worker per core.
var workers = runtime.NumCPU()
var userStateFile string

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())

  logfile   := flag.String("F", "/var/log/syslog", "Logfile to check")
  pattern   := flag.String("q", "ERROR", "Pattern to apply")
  warnStr   := flag.String("w", "1", "Warning Threshold")
  critStr   := flag.String("c", "1", "Critical Threshold")
  stateFStr := flag.String("O", "", "Alternate statefile location")
  invPat    := flag.String("i", "", "Pattern to ignore after match")

  flag.Parse()

  check := nagiosplugin.NewCheck()
  defer check.Finish()

  userStateFile = *stateFStr

  warningRange, err := nagiosplugin.ParseRange( *warnStr )
  if err != nil {
    check.AddResult(nagiosplugin.UNKNOWN, err.Error())
  }

  criticalRange, err := nagiosplugin.ParseRange( *critStr )
  if err != nil {
    check.AddResult(nagiosplugin.UNKNOWN, err.Error())
  }

  logfiles := globFiles( *logfile )
  found := checkLogs( logfiles, *pattern, *invPat )

  tpl := "Found %d matches for %v in %v"
  msg := fmt.Sprintf(tpl, found, *pattern, *logfile)
  if criticalRange.CheckUint64( found ) {
    check.AddResult(nagiosplugin.CRITICAL, msg)
  }
  if warningRange.CheckUint64( found ) {
    check.AddResult(nagiosplugin.WARNING, msg)
  }
  check.AddResult(nagiosplugin.OK, msg)
  check.AddPerfDatum("matches","n",float64(found), 0.0, math.Inf(1), 0.0, 0.0)
	// the defered check.Finish() should make sure that a valid
	// nrpe compatible output is generated no matter what
}

// globFiles will try to expand the logfile pattern
func globFiles(logfile string) []string {
  logfiles := make([]string, 0)
  matches, err := filepath.Glob(logfile)
  if err != nil {
    logfiles = append(logfiles, logfile)
  } else {
    logfiles = append(logfiles, matches...)
  }
  return logfiles
}

// checkLogs does the work of scanning the logfiles, handling state,
// concurrency and applying the patterns
func checkLogs(logfiles []string, pattern string, invPat string) uint64 {
  lines     := make(chan string, workers*4)
  done      := make(chan struct{}, workers)
  results   := make(chan uint64, 1000)

  go readLines(logfiles, lines, pattern, invPat)
  processLines(done, lines, results, pattern, invPat)
  go awaitCompletion(done, results)
  found := processResults( results )

  return found
}

// readLines is executed exactly once in its own goroutine.
// It reads the input file, checks the preconditions and then passes
// it to the lines channel which is read by the processLines functions.
func readLines(filenames []string, lines chan<- string, pattern string, invPat string) {
  state := LoadState(userStateFile)
  defer state.Save()

  FILE: for _, filename := range filenames {
    offset := state.GetOffset( filename, pattern, invPat )
    file, err := os.Open(filename)
    if err != nil {
      fmt.Println("failed to open the file:", err)
      continue FILE
    }
    // close the file when we return
    defer file.Close()

		// resume reading the file if we've already touched this file
		// otherwise the offset will be zero - see state
    _, err = file.Seek( offset, 0 )
    if err != nil {
      fmt.Println("failed to seek to position:", offset, err)
    }
    reader := bufio.NewReader(file)
    LINE: for {
      line, err := reader.ReadBytes('\n')
      line = bytes.TrimRight(line, "\n\r")
      if len(line) > 0 {
        lines <- string(line)
      }
      if err != nil {
        if err != io.EOF {
          fmt.Println("failed to finish reading the file:", err)
        }
        break LINE
      }
    }
    offset, err = file.Seek( 0, os.SEEK_CUR )
    if err != nil {
      fmt.Println("failed to get seek position:", err)
    }
		// remember where we stopped reading
    state.SetOffset( filename, pattern, invPat, offset )
		// important: close the file now, not (only) w/ defer
		// or you risk running out of file handles
    file.Close()
  }
  state.Save()
	// important: closing the lines channel must be the last line in this method
	// or the process may exit before the remaining work is done!
  close(lines)
}

// processLines pre-compiles the regexp.Regexp objects from the patterns and spawns one
// worker per slot available (usually one per core).
func processLines(done chan<- struct{}, lines <-chan string, results chan<- uint64, pattern string, invPat string) {
  lineRx := regexp.MustCompile(pattern)
  var invLineRx *regexp.Regexp
  if len(invPat) > 0 {
    invLineRx = regexp.MustCompile(invPat)
  }
  // set invLineRx to nil if invPat is empty
  for i := 0; i < workers; i++ {
    go processLine(done, lines, results, lineRx, invLineRx)
  }
}

// processLine reads from the lines channel, processing the lines as they are being read from disk,
// apply the positive regexp and possible the negative (ignore) regexp until the lines
// channel is closed
func processLine(done chan<- struct{}, lines <-chan string, results chan<- uint64, lineRx *regexp.Regexp, invLineRx *regexp.Regexp) {
  var found uint64 = 0
  // process each line from the lines channel
  for line := range lines {
    if lineRx.MatchString(line) {
      if invLineRx == nil || !invLineRx.MatchString(line) {
        found += 1
      }
    } // else lineRx didn't match
  }
  results <- found
  done <- struct{}{}
}

// awaitCompletion just waits until all workers have reported
// back so that the main goroutine does not exit until
// all work is done.
func awaitCompletion(done <-chan struct{}, results chan uint64) {
  for i := 0; i < workers; i++ {
    <-done
  }
  close(results)
}

// processResults just sums up the results reported by the individual
// worker goroutines.
func processResults(results <-chan uint64) (sumFound uint64) {
  // loop over results
  for result := range results {
    sumFound += result
  }
  return sumFound
}

