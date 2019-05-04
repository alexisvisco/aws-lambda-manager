package commands

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/spf13/cobra"
)

var FlagRegion string
var SessionAWS *session.Session

var RootCmd = &cobra.Command{
	Use:          "expected",
	Short:        "Expected cli is the fastest and efficient way to deploy on aws, gc ...",
	Long:         `Deploy on amazon, google cloud, azure in a way that is absolutely simple and efficient for you.`,
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			return
		}
	},
}

func init() {
	RootCmd.PersistentFlags().StringVar(&FlagRegion, "region", "eu-west-3", "region to use")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(FlagRegion),
	})
	if err != nil {
		return
	}
	SessionAWS = sess
}
