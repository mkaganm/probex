package test

import "os"

// OS function wrappers used by tests that need to change directories.
var (
	_getwd = os.Getwd
	_chdir = os.Chdir
)
