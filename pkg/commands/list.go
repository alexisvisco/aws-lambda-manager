package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"aws-test/pkg/amazon"
	"aws-test/pkg/util"

	"github.com/spf13/cobra"
)

// flListVersionFull show full sha256
var flListVersionFull bool

func listVersions(_ *cobra.Command, args []string) error {
	resourceName := fmt.Sprintf("%s-%s", args[0], args[1])
	output, err := amazon.S3ListObjects(awsSession, resourceName)
	if err != nil {
		return err
	}

	tab := tabwriter.NewWriter(os.Stdout, 1, 0, 4, ' ', 0)
	_, _ = fmt.Fprintf(tab, "SHA256 ID\tSIZE\tCREATED AT\t\n")
	for _, l := range output.Contents {
		split := strings.Split(*l.Key, "-")
		if len(split) == 2 {
			t := split[0]
			sha := split[1]

			i, err := strconv.ParseInt(t, 10, 64)
			if err != nil {
				return err
			}
			versionTime := time.Unix(i, 0)

			if flListVersionFull {
				_, _ = fmt.Fprintf(tab, "%s\t%s\t%s\t\n", sha[:len(sha)-4], util.HumanByteSize(*l.Size), t)
			} else {
				_, _ = fmt.Fprintf(tab, "%s\t%s\t%s\t\n", sha[:12], util.HumanByteSize(*l.Size), versionTime.Format(time.RFC822))
			}
		}
	}
	_ = tab.Flush()
	return nil
}

// flListAll lambdas even if awsl did not create them
var flListAll bool

func list(_ *cobra.Command, _ []string) error {
	list, err := amazon.LambdaGetAll(awsSession, flListAll)
	if err != nil {
		return err
	}

	tab := tabwriter.NewWriter(os.Stdout, 1, 0, 4, ' ', 0)
	_, _ = fmt.Fprintf(tab, "NAME\tID\tRUNTIME\tMEMORY\tARN\t\n")

	for _, f := range list {
		split := strings.Split(*f.FunctionName, "-")
		name := strings.Join(split[:len(split)-1], "-")
		_, _ = fmt.Fprintf(tab, "%s\t%s\t%s\t%s\t%s\t\n", name, split[len(split)-1], *f.Runtime, util.HumanByteSize(*f.MemorySize*1000000), *f.FunctionArn)
	}
	_ = tab.Flush()
	return nil
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List of lambdas",
		RunE:  list,
	}
	listCmd.PersistentFlags().BoolVarP(&flDeployForce, "all", "a", false, "list all lambdas even if awsl did not create them")

	listVersion := &cobra.Command{
		Use:   "list-version <name> <id>",
		Short: "List of version for a given lambda",
		Args:  cobra.ExactArgs(2),
		RunE:  listVersions,
	}
	listVersion.PersistentFlags().BoolVarP(&flListVersionFull, "full", "f", false, "show full sha256")

	Root.AddCommand(listCmd)
	Root.AddCommand(listVersion)
}
