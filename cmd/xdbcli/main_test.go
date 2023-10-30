package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	GitVersion = "0.1.1"
	out := bytes.NewBuffer([]byte{})
	errout := bytes.NewBuffer([]byte{})
	rc := 0
	exit := func(c int) {
		rc = c
	}

	realMain([]string{"xdbcli", "--version"}, out, errout, exit)
	assert.Equal(t, "0.1.1\n", out.String())
	// since our exit func does not call os.Exit, the next parser will fail
	assert.Equal(t, 1, rc)
	assert.NotEmpty(t, errout.String())
}
