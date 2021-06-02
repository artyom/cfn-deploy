// cfn-deploy creates or updates CloudFormation stack from a template file
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

func main() {
	log.SetFlags(0)
	args := runArgs{}
	flag.StringVar(&args.File, "f", args.File, "template `file`")
	flag.StringVar(&args.Name, "n", args.Name, "stack `name`")
	flag.BoolVar(&args.Create, "c", args.Create, "create stack instead of updating")
	flag.Func("t", "add a tag in `name=value` format; can be used multile times", func(s string) error {
		i := strings.IndexRune(s, '=')
		if i == -1 {
			return errors.New("tag must be in 'name=value' format")
		}
		name, value := strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
		if name == "" || value == "" {
			return errors.New("both tag name and value must be set")
		}
		args.Tags = append(args.Tags, types.Tag{Key: &name, Value: &value})
		return nil
	})
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx, args); err != nil {
		log.Fatal(err)
	}
}

type runArgs struct {
	File   string
	Name   string
	Create bool
	Tags   []types.Tag
}

func run(ctx context.Context, args runArgs) error {
	if args.Name == "" {
		return errors.New("stack name must be set")
	}
	body, err := os.ReadFile(args.File)
	if err != nil {
		return err
	}
	var capabilities []types.Capability
	if bytes.Contains(body, []byte("AWS::IAM::")) {
		capabilities = append(capabilities, types.CapabilityCapabilityNamedIam)
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	client := cloudformation.NewFromConfig(cfg)
	if args.Create {
		_, err = client.CreateStack(ctx, &cloudformation.CreateStackInput{
			StackName:    &args.Name,
			TemplateBody: aws.String(string(body)),
			OnFailure:    types.OnFailureDelete,
			Capabilities: capabilities,
			Tags:         args.Tags,
		})
		return err
	}
	_, err = client.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:    &args.Name,
		TemplateBody: aws.String(string(body)),
		Capabilities: capabilities,
		Tags:         args.Tags,
	})
	return err
}
