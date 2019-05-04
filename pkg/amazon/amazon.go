package amazon

import (
	"aws-test/pkg/util"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const lambdaAssumeRolePolicyDocument = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":["apigateway.amazonaws.com","logs.amazonaws.com","lambda.amazonaws.com"]},"Action":"sts:AssumeRole"}]}`

func S3BucketExist(sess *session.Session, bucketName string) bool {
	s := s3.New(sess)
	_, err := s.HeadBucket(&s3.HeadBucketInput{Bucket: aws.String(bucketName)})
	return err == nil
}

func S3CreateBucket(sess *session.Session, bucketName string) error {
	s := s3.New(sess)

	_, err := s.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	return err
}

func S3DeleteBucket(sess *session.Session, bucketName string) error {
	s := s3.New(sess)

	_, err := s.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	return err
}

func S3ListObjects(sess *session.Session, bucketName string) (*s3.ListObjectsOutput, error) {
	s := s3.New(sess)
	return s.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	})
}

func S3FileExist(sess *session.Session, bucketName string, sum string) bool {
	s := s3.New(sess)
	output, err := s.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return false
	}
	for _, content := range output.Contents {
		filename := *content.Key
		extension := filepath.Ext(filename)
		name := filename[0 : len(filename)-len(extension)]
		n := strings.SplitN(name, "-", 2)
		if len(n) == 2 {
			if n[1] == sum {
				return true
			}
		}
	}
	return false
}

func S3UploadFile(sess *session.Session, bucketName, sum, file string) (string, *s3manager.UploadOutput, error) {
	name := fmt.Sprintf("%d-%s.zip", time.Now().Unix(), sum)
	uploader := s3manager.NewUploader(sess)
	f, err := os.Open(file)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()

	output, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(name),
		Body:   f,
	})
	return name, output, err
}

func LambdaGet(sess *session.Session, name string) *lambda.GetFunctionOutput {
	l := lambda.New(sess)
	la, e := l.GetFunction(&lambda.GetFunctionInput{
		FunctionName: aws.String(name),
	})
	if e != nil {
		return nil
	} else {
		return la
	}
}

func LambdaGetAll(sess *session.Session, all bool) ([]*lambda.FunctionConfiguration, error) {
	l := lambda.New(sess)
	la, err := l.ListFunctions(&lambda.ListFunctionsInput{})
	if err != nil {
		return nil, err
	}
	if all {
		return la.Functions, nil
	}
	var list []*lambda.FunctionConfiguration
	for _, lam := range la.Functions {
		output, err := l.ListTags(&lambda.ListTagsInput{
			Resource: lam.FunctionArn,
		})
		if err != nil {
			return nil, err
		}
		i, ok := output.Tags["manager"]
		if ok && *i == "expected.sh" {
			list = append(list, lam)
		}
	}
	return list, nil
}

func LambdaCreate(sess *session.Session, name, s3Key string) (link *string, err error) {
	var cfg *lambda.FunctionConfiguration

	output, _ := iam.New(sess).GetUser(&iam.GetUserInput{})
	accountId := strings.Split(*output.User.Arn, ":")[4]

	l := lambda.New(sess)
	rolesOutput, err := iam.New(sess).CreateRole(&iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(lambdaAssumeRolePolicyDocument),
		MaxSessionDuration:       aws.Int64(3600),
		Path:                     aws.String("/service-role/"),
		RoleName:                 aws.String(name),
	})
	if err != nil {
		return nil, err
	}

	time.Sleep(3 * time.Second)

	err = util.NewBackoff("create function", func() error {
		cfg, err = l.CreateFunction(&lambda.CreateFunctionInput{
			Code: &lambda.FunctionCode{
				S3Bucket: aws.String(name),
				S3Key:    aws.String(s3Key),
			},
			Role:         rolesOutput.Role.Arn,
			FunctionName: aws.String(name),
			Handler:      aws.String("main"),
			MemorySize:   aws.Int64(256),
			Publish:      aws.Bool(true),
			Runtime:      aws.String("go1.x"),
			Tags: map[string]*string{
				"manager": aws.String("expected.sh"),
			},
			Timeout: aws.Int64(15),
		})
		return err
	}).Execute()

	if err != nil {
		return nil, err
	}

	gateway := apigateway.New(sess)

	api, errx := gateway.CreateRestApi(&apigateway.CreateRestApiInput{
		ApiKeySource:     aws.String("HEADER"),
		BinaryMediaTypes: []*string{aws.String("*/*")},
		EndpointConfiguration: &apigateway.EndpointConfiguration{
			Types: []*string{aws.String("REGIONAL")},
		},
		Name: aws.String(fmt.Sprintf("%s-API", name)),
	})
	if errx != nil {
		return nil, errx
	}

	resources, errx := gateway.GetResources(&apigateway.GetResourcesInput{
		RestApiId: api.Id,
	})

	if errx != nil {
		return nil, errx
	}

	if len(resources.Items) != 1 {
		return nil, errors.New("bad api gateway construction")
	}

	parentResourceId := resources.Items[0].Id

	resource, errx := gateway.CreateResource(&apigateway.CreateResourceInput{
		ParentId:  parentResourceId,
		PathPart:  aws.String(name),
		RestApiId: api.Id,
	})

	if errx != nil {
		return nil, errx
	}

	_, errx = gateway.PutMethod(&apigateway.PutMethodInput{
		ApiKeyRequired:    aws.Bool(false),
		AuthorizationType: aws.String("NONE"),
		HttpMethod:        aws.String("ANY"),
		ResourceId:        resource.Id,
		RestApiId:         api.Id,
	})

	if errx != nil {
		return nil, errx
	}

	_, errx = gateway.PutIntegration(&apigateway.PutIntegrationInput{
		HttpMethod:            aws.String("ANY"),
		IntegrationHttpMethod: aws.String("POST"),
		PassthroughBehavior:   aws.String("WHEN_NO_MATCH"),
		ResourceId:            resource.Id,
		RestApiId:             api.Id,
		TimeoutInMillis:       aws.Int64(29000),
		Type:                  aws.String("AWS_PROXY"),
		Uri:                   aws.String(fmt.Sprintf("arn:aws:apigateway:eu-west-3:lambda:path/2015-03-31/functions/%s/invocations", *cfg.FunctionArn)),
	})

	if errx != nil {
		return nil, errx
	}

	_, errx = gateway.PutIntegrationResponse(&apigateway.PutIntegrationResponseInput{
		HttpMethod:       aws.String("ANY"),
		ResourceId:       resource.Id,
		RestApiId:        api.Id,
		SelectionPattern: aws.String(".*"),
		StatusCode:       aws.String("200"),
	})

	if errx != nil {
		return nil, errx
	}

	_, errx = gateway.PutMethodResponse(&apigateway.PutMethodResponseInput{
		HttpMethod: aws.String("ANY"),
		ResourceId: resource.Id,
		RestApiId:  api.Id,
		StatusCode: aws.String("200"),
	})
	if errx != nil {
		return nil, errx
	}

	_, errx = gateway.CreateDeployment(&apigateway.CreateDeploymentInput{
		Description: aws.String("Created by Expected.sh"),
		RestApiId:   api.Id,
		StageName:   aws.String("default"),
	})
	if errx != nil {
		return nil, errx
	}

	_, errx = l.AddPermission(&lambda.AddPermissionInput{
		Action:       aws.String("lambda:InvokeFunction"),
		Principal:    aws.String("apigateway.amazonaws.com"),
		FunctionName: cfg.FunctionName,
		SourceArn: aws.String(fmt.Sprintf("arn:aws:execute-api:%s:%s:%s/*/*/%s",
			*sess.Config.Region, accountId, *api.Id, name,
		)),
		StatementId: api.Name,
	})
	if errx != nil {
		return nil, errx
	}

	lambdaLink := fmt.Sprintf("https://%s.execute-api.%s.amazonaws.com/default/%s", *api.Id, *sess.Config.Region, name)
	return &lambdaLink, err
}

func LambdaUpdateCode(sess *session.Session, name, s3Key string) (*lambda.FunctionConfiguration, error) {
	l := lambda.New(sess)
	return l.UpdateFunctionCode(&lambda.UpdateFunctionCodeInput{
		FunctionName: aws.String(name),
		Publish:      aws.Bool(true),
		S3Bucket:     aws.String(name),
		S3Key:        aws.String(s3Key),
	})
}

func LambdaDelete(sess *session.Session, name string) error {
	_, err := iam.New(sess).DeleteRole(&iam.DeleteRoleInput{
		RoleName: aws.String(name),
	})
	if err != nil {
		return err
	}
	_, err = lambda.New(sess).DeleteFunction(&lambda.DeleteFunctionInput{
		FunctionName: aws.String(name),
	})
	if err != nil {
		return err
	}

	gateway := apigateway.New(sess)
	l, err := gateway.GetRestApis(&apigateway.GetRestApisInput{})
	if err == nil {
		for _, i := range l.Items {
			if *i.Name == fmt.Sprintf("%s-API", name) {
				_, err = gateway.DeleteRestApi(&apigateway.DeleteRestApiInput{
					RestApiId: i.Id,
				})
				if err != nil {
					return err
				}
			}
		}
	} else {
		return err
	}
	return nil
}
