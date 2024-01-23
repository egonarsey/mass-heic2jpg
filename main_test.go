package main

import (
	"testing"
)

func Test_getOldHeicDir(t *testing.T) {
	args := []string{"./testdir", "./testdir"}
	oldHeicDir, err := getOldHeicDir(args)
	if oldHeicDir != "./testdir" || err != nil {
		t.Fatalf(`getOldHeicDir("") %q %v`, oldHeicDir, err)
	}
}
