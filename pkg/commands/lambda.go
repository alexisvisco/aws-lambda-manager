package commands

import (
	"aws-test/pkg/amazon"
	"aws-test/pkg/util"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

type lambdaCtx struct {
	folder, name, id string
}

var flagDeployForce bool
var flagDeployId string
var flagDeployRuntime string

func deploy(_ *cobra.Command, args []string) error {
	name := args[0]
	folder := args[1]

	if _, err := os.Stat(folder); os.IsNotExist(err) {
		return err
	}

	lambdaCtx := lambdaCtx{folder: folder, name: name}
	if flagDeployId == "" {
		lambdaCtx.id = util.RandID(12)
	} else {
		lambdaCtx.id = flagDeployId
	}

	var (
		sum, s3key string
		file       *os.File
		link       *string
		err        error
	)

	resourceName := fmt.Sprintf("%s-%s", lambdaCtx.name, lambdaCtx.id)

	// Bucket creation to store code
	if !amazon.S3BucketExist(SessionAWS, resourceName) {
		if err := util.Action(fmt.Sprintf("Creating bucket %s", resourceName), func() error {
			return amazon.S3CreateBucket(SessionAWS, resourceName)
		}); err != nil {
			return err
		}
	}

	// Create a local zip of the code in the folder
	if err := util.Action(fmt.Sprintf("Creating zip of your code"), func() error {
		sum, file, err = util.CreateZip(lambdaCtx.folder)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	// Upload the code on provider
	if err := util.Action(fmt.Sprintf("Uploading your lambda with sum %s to s3", sum), func() error {
		if !flagDeployForce && amazon.S3FileExist(SessionAWS, resourceName, sum) {
			return errors.New("lambda with this version already exist")
		}
		s3key, _, err = amazon.S3UploadFile(SessionAWS, resourceName, sum, file.Name())
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	// Create or Update the lambda
	lambdaGet := amazon.LambdaGet(SessionAWS, resourceName)
	if lambdaGet != nil {
		if err := util.Action(fmt.Sprintf("Updating code of your lambda"), func() error {
			_, err := amazon.LambdaUpdateCode(SessionAWS, resourceName, s3key)
			return err
		}); err != nil {
			return err
		}
	} else {
		if err := util.Action(fmt.Sprintf("Creating your lambda"), func() error {
			link, err = amazon.LambdaCreate(SessionAWS, lambdaCtx.id, flagDeployRuntime, resourceName, s3key)
			return err
		}); err != nil {
			return err
		}
	}

	fmt.Println("Lambda id   ", lambdaCtx.id)
	if link != nil {
		fmt.Println("Lambda link ", *link)
	}

	return nil
}

var flagRollbackTime string

func rollback(_ *cobra.Command, args []string) error {
	resourceName := fmt.Sprintf("%s-%s", args[0], args[1])
	output, err := amazon.S3ListObjects(SessionAWS, resourceName)
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
	} else if len(list) > 1 && flagRollbackTime == "" {
		listOfVersions := ""
		for _, obj := range list {
			split := strings.Split(*obj.Key, "-")
			listOfVersions += fmt.Sprintf(" - time: %s	sha256: %s\n", split[0], split[1][:len(split[1])-4])
		}
		return errors.New("multiple versions founded, use -time option to specify the exact sha256 you want among:\n" + listOfVersions)
	} else {
		var target *s3.Object = nil
		if len(list) == 1 {
			target = list[0]
		} else {
			for _, obj := range list {
				split := strings.Split(*obj.Key, "-")
				if split[0] == flagRollbackTime {
					target = obj
				}
			}
		}

		if target == nil {
			return errors.New("no versions founded")
		}

		split := strings.Split(*target.Key, "-")

		if err := util.Action(fmt.Sprintf("Rollback to version %s", split[1][:len(split[1])-4]), func() error {
			_, err := amazon.LambdaUpdateCode(SessionAWS, resourceName, *target.Key)
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}

var flagRemoveStorage bool

func remove(_ *cobra.Command, args []string) error {
	if err := amazon.LambdaDelete(SessionAWS, fmt.Sprintf("%s-%s", args[0], args[1])); err != nil {
		return err
	}
	if flagRemoveStorage {
		return amazon.S3DeleteBucket(SessionAWS, fmt.Sprintf("%s-%s", args[0], args[1]))
	}
	return nil
}

var flagListVersionFull bool

func listVersions(_ *cobra.Command, args []string) error {
	resourceName := fmt.Sprintf("%s-%s", args[0], args[1])
	output, err := amazon.S3ListObjects(SessionAWS, resourceName)
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

			if flagListVersionFull {
				_, _ = fmt.Fprintf(tab, "%s\t%s\t%s\t\n", sha[:len(sha)-4], util.HumanByteSize(*l.Size), t)
			} else {
				_, _ = fmt.Fprintf(tab, "%s\t%s\t%s\t\n", sha[:12], util.HumanByteSize(*l.Size), versionTime.Format(time.RFC822))
			}
		}
	}
	_ = tab.Flush()
	return nil
}

var flagListAll bool

func list(_ *cobra.Command, _ []string) error {
	list, err := amazon.LambdaGetAll(SessionAWS, flagListAll)
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
	lambdaCmd := &cobra.Command{
		Use:   "lambda",
		Short: "Manage deployment for a lambda",
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				return
			}
		},
	}

	lambdaCmdDeploy := &cobra.Command{
		Use:   "deploy <name> <folder>",
		Short: "Create or update a lambda",
		Args:  cobra.ExactArgs(2),
		RunE:  deploy,
	}
	lambdaCmdDeploy.PersistentFlags().BoolVarP(&flagDeployForce, "force", "f", false, "force deployment if code already exist")
	lambdaCmdDeploy.PersistentFlags().StringVar(&flagDeployId, "id", "", "set the id of the lambda, if none a new lambda will be created")
	lambdaCmdDeploy.PersistentFlags().StringVarP(&flagDeployRuntime, "runtime", "r", "go1.x", "set the runtime (the programming language) of the function")

	lambdaCmdRollback := &cobra.Command{
		Use:   "rollback <name> <id> <sha256 version>",
		Short: "Rollback a lambda to a certain version",
		Args:  cobra.ExactArgs(3),
		RunE:  rollback,
	}
	lambdaCmdRollback.PersistentFlags().StringVarP(&flagRollbackTime, "time", "t", "", "use a time versioned sha256")

	lambdaCmdDelete := &cobra.Command{
		Use:   "delete",
		Short: "Delete a lambda",
		Args:  cobra.ExactArgs(2),
		RunE:  remove,
	}
	lambdaCmdDelete.PersistentFlags().BoolVarP(&flagRemoveStorage, "storage", "s", true, "remove bucket where code is stored")

	lambdaCmdList := &cobra.Command{
		Use:   "list",
		Short: "List of lambdas",
		RunE:  list,
	}
	lambdaCmdList.PersistentFlags().BoolVarP(&flagDeployForce, "all", "a", false, "list all lambdas even if expected.sh did not create them")

	lambdaCmdListVersion := &cobra.Command{
		Use:   "list-version <name> <id>",
		Short: "List of version for a given lambda",
		Args:  cobra.ExactArgs(2),
		RunE:  listVersions,
	}
	lambdaCmdListVersion.PersistentFlags().BoolVarP(&flagListVersionFull, "full", "f", false, "show full sha256")

	RootCmd.AddCommand(lambdaCmd)

	lambdaCmd.AddCommand(lambdaCmdDeploy)
	lambdaCmd.AddCommand(lambdaCmdRollback)
	lambdaCmd.AddCommand(lambdaCmdDelete)
	lambdaCmd.AddCommand(lambdaCmdList)
	lambdaCmd.AddCommand(lambdaCmdListVersion)
}
