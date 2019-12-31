package commands

import (
	"fmt"

	"aws-test/pkg/amazon"

	"github.com/spf13/cobra"
)

var flRemoveStorage bool

func remove(_ *cobra.Command, args []string) error {
	if err := amazon.LambdaDelete(awsSession, fmt.Sprintf("%s-%s", args[0], args[1])); err != nil {
		return err
	}
	if flRemoveStorage {
		return amazon.S3DeleteBucket(awsSession, fmt.Sprintf("%s-%s", args[0], args[1]))
	}
	return nil
}

func init() {
	cmdRemove := &cobra.Command{
		Use:   "remove",
		Short: "Remove a lambda",
		Args:  cobra.ExactArgs(2),
		RunE:  remove,
	}
	cmdRemove.PersistentFlags().BoolVarP(&flRemoveStorage, "storage", "s", true, "remove bucket where code is stored")

	Root.AddCommand(cmdRemove)
}
