package amazon

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"aws-test/pkg/util"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/lambda"
)

const lambdaAssumeRolePolicyDocument = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":["apigateway.amazonaws.com","logs.amazonaws.com","lambda.amazonaws.com"]},"Action":"sts:AssumeRole"}]}`

type Function struct {
	*lambda.FunctionConfiguration
	Tags map[string]*string
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

func LambdaGetAll(sess *session.Session, all bool) ([]Function, error) {
	l := lambda.New(sess)
	la, err := l.ListFunctions(&lambda.ListFunctionsInput{})
	if err != nil {
		return nil, err
	}
	var list []Function
	for _, lam := range la.Functions {
		output, err := l.ListTags(&lambda.ListTagsInput{
			Resource: lam.FunctionArn,
		})
		if err != nil {
			return nil, err
		}
		if all {
			list = append(list, Function{lam, output.Tags})
			continue
		}
		i, ok := output.Tags["manager"]
		if ok && *i == "awsl" {
			list = append(list, Function{lam, output.Tags})
		}
	}
	return list, nil
}

func LambdaCreate(sess *session.Session, id, runtime, name, s3Key string) (link *string, err error) {
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
			Runtime:      aws.String(runtime),
			Tags: map[string]*string{
				"manager": aws.String("awsl"),
				"created": aws.String(fmt.Sprintf("%d", time.Now().Unix())),
				"id":      aws.String(id),
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
		Description: aws.String("Created by awsl"),
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
