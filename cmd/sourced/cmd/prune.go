package cmd

import (
	"context"

	"github.com/src-d/sourced-ce/cmd/sourced/compose"
)

type pruneCmd struct {
	Command `name:"prune" short-description:"Stop and remove containers and resources" long-description:"Stops containers and removes containers, networks, and volumes created by 'init'.\nImages are not deleted unless you specify the --images flag."`

	Images bool `long:"images" description:"Remove docker images"`
}

func (c *pruneCmd) Execute(args []string) error {
	a := []string{"down", "--volumes"}
	if c.Images {
		a = append(a, "--rmi", "all")
	}

	return compose.Run(context.Background(), a...)
}

func init() {
	rootCmd.AddCommand(&pruneCmd{})
}
