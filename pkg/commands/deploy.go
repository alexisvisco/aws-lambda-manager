package commands

import (
	"errors"
	"fmt"
	"os"

	"aws-test/pkg/amazon"
	"aws-test/pkg/util"

	"github.com/spf13/cobra"
)

// flDeployForce force deployment if code already exist
var flDeployForce bool

// flDeployId set the id of the lambda, if none a new lambda will be created
var flDeployId string

// flDeployRuntime set the runtime (the programming language) of the function
var flDeployRuntime string

func deploy(_ *cobra.Command, args []string) error {
	name := args[0]
	folder := args[1]

	fmt.Println(folder)

	if _, err := os.Stat(folder); os.IsNotExist(err) {
		return err
	}

	lambdaCtx := lambdaCtx{folder: folder, name: name}
	if flDeployId == "" {
		lambdaCtx.id = util.RandID(12)
	} else {
		lambdaCtx.id = flDeployId
	}

	var (
		sum, s3key string
		file       *os.File
		link       *string
		err        error
	)

	resourceName := fmt.Sprintf("%s-%s", lambdaCtx.name, lambdaCtx.id)

	// Bucket creation to store code
	if !amazon.S3BucketExist(awsSession, resourceName) {
		if err := util.Action(fmt.Sprintf("Creating bucket %s", resourceName), func() error {
			return amazon.S3CreateBucket(awsSession, resourceName)
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

	// Upload the code on the provider
	if err := util.Action(fmt.Sprintf("Uploading your lambda with sum %s to s3", sum), func() error {
		if !flDeployForce && amazon.S3FileExist(awsSession, resourceName, sum) {
			return errors.New("lambda with this version already exist")
		}
		s3key, _, err = amazon.S3UploadFile(awsSession, resourceName, sum, file.Name())
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	// Create or Update the lambda
	lambdaGet := amazon.LambdaGet(awsSession, resourceName)
	if lambdaGet != nil {
		if err := util.Action(fmt.Sprintf("Updating code of your lambda"), func() error {
			_, err := amazon.LambdaUpdateCode(awsSession, resourceName, s3key)
			return err
		}); err != nil {
			return err
		}
	} else {
		if err := util.Action(fmt.Sprintf("Creating your lambda"), func() error {
			link, err = amazon.LambdaCreate(awsSession, lambdaCtx.id, flDeployRuntime, resourceName, s3key)
			return err
		}); err != nil {
			return err
		}
	}

	fmt.Println("Lambda id   ", lambdaCtx.id)
	if link != nil {
		fmt.Println("Lambda public link ", *link)
	}

	return nil
}

func init() {
	cmdDeploy := &cobra.Command{
		Use:   "deploy <name> <folder>",
		Short: "Create or update a lambda",
		Args:  cobra.ExactArgs(2),
		RunE:  deploy,
	}
	cmdDeploy.PersistentFlags().BoolVarP(&flDeployForce, "force", "f", false, "force deployment if code already exist")
	cmdDeploy.PersistentFlags().StringVar(&flDeployId, "id", "", "set the id of the lambda, if none a new lambda will be created")
	cmdDeploy.PersistentFlags().StringVarP(&flDeployRuntime, "runtime", "r", "go1.x", "set the runtime (the programming language) of the function")

	Root.AddCommand(cmdDeploy)
}
