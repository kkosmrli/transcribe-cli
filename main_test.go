package main

import (
	"os"
	"os/exec"
	"testing"
)

// TestCheckFileTypeInvalid tests the CheckFileType function with an invalid file type.
// It expects the program to exit with a status code of 1.
func TestCheckFileTypeInvalid(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		CheckFileType("test.txt")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestCheckFileTypeInvalid")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return // Program exited with non-zero status, as expected.
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

// TestCheckFileTypeValid tests the CheckFileType function with a valid file type.
// It expects the function to run without error.
func TestCheckFileTypeValid(t *testing.T) {
	// This test simply calls CheckFileType with a valid name.
	// If CheckFileType calls log.Fatalf, this test will fail.
	CheckFileType("test.m4a")
}
