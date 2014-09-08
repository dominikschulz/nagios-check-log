// +build !windows
package main

import (
  "os"
  "syscall"
)

func NewFileState(path string) (*FileState, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	fstat := info.Sys().(*syscall.Stat_t)
	return &FileState{
		Source: path,
		Size: info.Size(),
		Inode: fstat.Ino,
		Device: fstat.Dev,
	}, nil
}

func (state *FileState) SameFile(path string) bool {
	other, err := NewFileState(path)
  if err != nil {
    return false
  }
	return state.SameState(other)
}

func (state *FileState) SameState(other *FileState) bool {
	if other == nil {
		return false
	}
	if other.Size < state.Size {
		return false
	}
	return (other.Inode == state.Inode && other.Device == state.Device)
}

