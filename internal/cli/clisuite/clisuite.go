// Package clisuite to test CLI commands
package clisuite

import (
	"bytes"
	"os"
	"path"

	"github.com/alecthomas/kong"
	"github.com/effective-security/x/ctl"
	"github.com/effective-security/xdb/internal/cli"
	"github.com/stretchr/testify/suite"
)

// TestSuite provides suite for testing HTTP
type TestSuite struct {
	suite.Suite

	Folder string
	Ctl    *cli.Cli
	// Out is the outpub buffer
	Out bytes.Buffer
}

// HasText is a helper method to assert that the out stream contains the supplied
// text somewhere
func (s *TestSuite) HasText(texts ...string) {
	outStr := s.Out.String()
	for _, t := range texts {
		s.Contains(outStr, t)
	}
}

// EqualOut is a helper method to assert that the out stream contains the supplied
func (s *TestSuite) EqualOut(text string) {
	s.Equal(text, s.Out.String())
}

// HasNoText is a helper method to assert that the out stream does contains the supplied
// text somewhere
func (s *TestSuite) HasNoText(texts ...string) {
	outStr := s.Out.String()
	for _, t := range texts {
		s.Contains(outStr, t)
	}
}

// SetupSuite called once to setup
func (s *TestSuite) SetupSuite() {
	s.Folder = path.Join(os.TempDir(), "test", "xdb")

	s.Ctl = &cli.Cli{
		Version: ctl.VersionFlag("0.1.1"),
	}

	s.Ctl.WithErrWriter(&s.Out).
		WithWriter(&s.Out)

	parser, err := kong.New(s.Ctl,
		kong.Name("testcli"),
		kong.Description("test CLI"),
		kong.Writers(&s.Out, &s.Out),
		ctl.BoolPtrMapper,
		//kong.Exit(exit),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{})
	if err != nil {
		s.FailNow("unexpected error constructing Kong: %+v", err)
	}

	_, err = parser.Parse([]string{
		"-D",
		"--sql-source", "postgres://postgres:postgres@127.0.0.1:15433?sslmode=disable",
	})
	if err != nil {
		s.FailNow("unexpected error parsing: %+v", err)
	}
}

// TearDownSuite called once to destroy
func (s *TestSuite) TearDownSuite() {
	_ = os.RemoveAll(s.Folder)
}

// SetupTest called before each test
func (s *TestSuite) SetupTest() {
	s.Out.Reset()
	s.Ctl.O = ""
}

// TearDownTest called after each test
func (s *TestSuite) TearDownTest() {
}
