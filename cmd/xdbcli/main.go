package main

import (
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/effective-security/xdb/pkg/cli"
	"github.com/effective-security/xdb/pkg/cli/schema"
	"github.com/effective-security/xpki/x/ctl"
)

// version is set by the build script
const version = "0.2.8"

type app struct {
	cli.Cli

	Schema schema.Cmd `cmd:"" help:"SQL schema commands"`
}

func main() {
	realMain(os.Args, os.Stdout, os.Stderr, os.Exit)
}

func realMain(args []string, out io.Writer, errout io.Writer, exit func(int)) {
	cl := app{
		Cli: cli.Cli{
			Version: ctl.VersionFlag(version),
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
			"version": version,
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
