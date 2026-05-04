package main

import (
	"github.com/christopherdang/vibecloud/cli/cmd"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	cmd.SetVersion(version, commit)
	cmd.Execute()
}
