package command

import (
	"os"

	"github.com/urfave/cli"
	"go.step.sm/cli-utils/step"
	"go.step.sm/cli-utils/usage"
)

var cmds []cli.Command
var currentContext *cli.Context

func init() {
	os.Unsetenv(step.IgnoreEnvVar)
	cmds = []cli.Command{
		usage.HelpCommand(),
	}
}

// Register adds the given command to the global list of commands.
// It sets recursively the command Flags environment variables.
func Register(c cli.Command) {
	step.SetEnvVar(&c)
	cmds = append(cmds, c)
}

// Retrieve returns all commands
func Retrieve() []cli.Command {
	return cmds
}

// ActionFunc returns a cli.ActionFunc that stores the context.
func ActionFunc(fn cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		currentContext = ctx
		return fn(ctx)
	}
}

// IsForce returns if the force flag was passed
func IsForce() bool {
	return currentContext != nil && currentContext.Bool("force")
}
