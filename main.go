package main

import (
	"context"
	"flag"
	"os"

	"github.com/cloudfoundry/cnb2cf/commands"
	"github.com/google/subcommands"
)

func main() {
	subcommands.Register(&commands.Package{}, "")

	flag.Parse()
	os.Exit(int(subcommands.Execute(context.Background())))
}
