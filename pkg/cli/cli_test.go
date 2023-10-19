package cli

import (
	"os"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/effective-security/xpki/x/ctl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContext(t *testing.T) {
	var c Cli

	assert.NotNil(t, c.ErrWriter())
	assert.NotNil(t, c.Writer())
	assert.NotNil(t, c.Reader())

	c.WithErrWriter(os.Stderr)
	c.WithReader(os.Stdin)
	c.WithWriter(os.Stdout)

	assert.NotNil(t, c.ErrWriter())
	assert.NotNil(t, c.Writer())
	assert.NotNil(t, c.Reader())
	assert.NotNil(t, c.Context())
}

func TestParse(t *testing.T) {
	t.Run("after_apply", func(t *testing.T) {
		t.Setenv("XDB_DATASOURCE", "")
		var cl struct {
			Cli
			Cmd struct{} `kong:"cmd"`
		}
		p := mustNew(t, &cl)
		_, err := p.Parse([]string{
			"cmd",
			"-D",
		})
		assert.EqualError(t, err, `missing flags: --provider=STRING`)
	})

	t.Run("default", func(t *testing.T) {
		t.Setenv("XDB_DATASOURCE", "test")
		var cl struct {
			Cli
			Cmd struct{} `kong:"cmd"`
		}
		p := mustNew(t, &cl)
		ctx, err := p.Parse([]string{
			"cmd",
			"--provider", "postgres",
		})
		require.NoError(t, err)
		assert.Equal(t, "test", cl.SQLSource)
		require.Equal(t, "cmd", ctx.Command())
	})

	t.Run("with params", func(t *testing.T) {
		t.Setenv("XDB_DATASOURCE", "test")
		var cl struct {
			Cli
			Cmd checkCfgCmd `kong:"cmd"`
		}
		p := mustNew(t, &cl)
		_, err := p.Parse([]string{
			"cmd",
			"--sql-source", "sqlserver://127.0.0.1?user id=sa&password=notused",
			"--provider", "postgres",
		})
		require.NoError(t, err)
		assert.Equal(t, "sqlserver://127.0.0.1?user id=sa&password=notused", cl.SQLSource)
	})
}

type checkCfgCmd struct {
}

func (c *checkCfgCmd) Run(ctx *kong.Context, cli *Cli) error {
	return nil
}

func mustNew(t *testing.T, cli any, options ...kong.Option) *kong.Kong {
	t.Helper()
	options = append([]kong.Option{
		kong.Name("test"),
		kong.Exit(func(int) {
			t.Helper()
			t.Fatalf("unexpected exit()")
		}),
		ctl.BoolPtrMapper,
	}, options...)
	parser, err := kong.New(cli, options...)
	require.NoError(t, err)

	return parser
}
