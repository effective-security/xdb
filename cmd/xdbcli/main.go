package main

import (
	"io"
	"os"
	"runtime/debug"

	"github.com/alecthomas/kong"
	"github.com/effective-security/x/ctl"
	"github.com/effective-security/xdb/internal/cli"
	"github.com/effective-security/xdb/internal/cli/schema"
)

// version is set by the build script: -ldflags "-X main.version=..."
var version string

// buildVersion returns the version set at build time, or the module version
// of a `go install ...@version` build.
func buildVersion() string {
	if version != "" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "devel"
}

type app struct {
	cli.Cli

	Schema schema.Cmd `cmd:"" help:"SQL schema commands"`
}

func main() {
	realMain(os.Args, os.Stdout, os.Stderr, os.Exit)
}

func realMain(args []string, out io.Writer, errout io.Writer, exit func(int)) {
	ver := buildVersion()
	cl := app{
		Cli: cli.Cli{
			Version: ctl.VersionFlag(ver),
		},
	}
	defer cl.Close()

	cl.Cli.WithErrWriter(errout).
		WithWriter(out)

	parser, err := kong.New(&cl,
		kong.Name("xdbcli"),
		kong.Description("SQL schema tool"),
		//kong.UsageOnError(),
		kong.Writers(out, errout),
		kong.Exit(exit),
		ctl.BoolPtrMapper,
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": ver,
		})
	if err != nil {
		panic(err)
	}

	ctx, err := parser.Parse(args[1:])
	parser.FatalIfErrorf(err)

	if ctx != nil {
		err = ctx.Run(&cl.Cli)
		ctx.FatalIfErrorf(err)
	}
}
