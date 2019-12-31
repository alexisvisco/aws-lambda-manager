package commands

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/spf13/cobra"
)

type lambdaCtx struct {
	folder, name, id string
}

// flRegion is the region to use
var flRegion string

// awsSession is the aws session used
var awsSession *session.Session

var Root = &cobra.Command{
	Use:   "awsl",
	Short: "awls cli is the fastest and efficient way to deploy on aws",
	Long: `Deploy on amazon in a way that is absolutely simple and efficient for you.
Included: 
 - Versions: using digest
 - Efficient storage: using s3 and zip your lambda
 - AWS Gateway setup`,
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			return
		}
	},
}

func init() {
	Root.PersistentFlags().StringVar(&flRegion, "region", "eu-west-3", "region to use")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(flRegion),
	})
	if err != nil {
		return
	}
	awsSession = sess
}
