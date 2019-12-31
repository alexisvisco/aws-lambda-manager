package commands

import (
	"errors"
	"fmt"
	"strings"

	"aws-test/pkg/amazon"
	"aws-test/pkg/util"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/cobra"
)

// flRollbackTime use a time versioned sha256
var flRollbackTime string

func rollback(_ *cobra.Command, args []string) error {
	resourceName := fmt.Sprintf("%s-%s", args[0], args[1])
	output, err := amazon.S3ListObjects(awsSession, resourceName)
	if err != nil {
		return err
	}

	var list []*s3.Object
	for _, l := range output.Contents {
		split := strings.Split(*l.Key, "-")
		if len(split) == 2 {
			if strings.HasPrefix(split[1], args[2]) {
				list = append(list, l)
			}
		}
	}

	if len(list) == 0 {
		return errors.New("no versions founded")
	} else if len(list) > 1 && flRollbackTime == "" {
		listOfVersions := ""
		for _, obj := range list {
			split := strings.Split(*obj.Key, "-")
			listOfVersions += fmt.Sprintf(" - time: %s	sha256: %s\n", split[0], split[1][:len(split[1])-4])
		}
		return errors.New("multiple versions found, use -time option to specify the exact sha256 you want among:\n" + listOfVersions)
	} else {
		var target *s3.Object = nil
		if len(list) == 1 {
			target = list[0]
		} else {
			for _, obj := range list {
				split := strings.Split(*obj.Key, "-")
				if split[0] == flRollbackTime {
					target = obj
				}
			}
		}

		if target == nil {
			return errors.New("no versions founded")
		}

		split := strings.Split(*target.Key, "-")

		if err := util.Action(fmt.Sprintf("Rollback to version %s", split[1][:len(split[1])-4]), func() error {
			_, err := amazon.LambdaUpdateCode(awsSession, resourceName, *target.Key)
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	cmdRollback := &cobra.Command{
		Use:   "rollback <name> <id> <sha256 version>",
		Short: "Rollback a lambda to a certain version",
		Args:  cobra.ExactArgs(3),
		RunE:  rollback,
	}
	cmdRollback.PersistentFlags().StringVarP(&flRollbackTime, "time", "t", "", "use a time versioned sha256")

	Root.AddCommand(cmdRollback)
}
