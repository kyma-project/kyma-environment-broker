package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	interval = time.Second
	retries  = 5
)

type ClientFactory interface {
	New(ctx context.Context, accessKeyID, secretAccessKey, region string) (Client, error)
}

type Client interface {
	AvailableZones(ctx context.Context, machineType string) ([]string, error)
	AvailableZonesCount(ctx context.Context, machineType string) (int, error)
}

type EC2API interface {
	DescribeInstanceTypeOfferings(ctx context.Context, params *ec2.DescribeInstanceTypeOfferingsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypeOfferingsOutput, error)
}

func NewFactory() ClientFactory {
	return AWSClientFactory{}
}

type AWSClientFactory struct{}

func (AWSClientFactory) New(ctx context.Context, accessKeyID, secretAccessKey, region string) (Client, error) {
	return NewClient(ctx, accessKeyID, secretAccessKey, region)
}

type AWSClient struct {
	ec2Client EC2API
}

func NewClient(ctx context.Context, key, secret, region string) (*AWSClient, error) {
	cfg, err := newAWSConfig(ctx, key, secret, region)
	if err != nil {
		return nil, fmt.Errorf("while creating AWS config: %w", err)
	}
	return &AWSClient{ec2Client: ec2.NewFromConfig(cfg)}, nil
}

func (c *AWSClient) AvailableZones(ctx context.Context, machineType string) ([]string, error) {
	params := &ec2.DescribeInstanceTypeOfferingsInput{
		LocationType: "availability-zone",
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-type"),
				Values: []string{machineType},
			},
		},
	}

	var resp *ec2.DescribeInstanceTypeOfferingsOutput
	var err error
	for i := 0; i < retries; i++ {
		resp, err = c.ec2Client.DescribeInstanceTypeOfferings(ctx, params)
		if err == nil {
			break
		}
		time.Sleep(interval)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to describe offerings: %w", err)
	}

	zones := make([]string, 0, len(resp.InstanceTypeOfferings))
	for _, offering := range resp.InstanceTypeOfferings {
		if offering.Location != nil {
			zones = append(zones, *offering.Location)
		}
	}

	return zones, nil
}

func (c *AWSClient) AvailableZonesCount(ctx context.Context, machineType string) (int, error) {
	zones, err := c.AvailableZones(ctx, machineType)
	if err != nil {
		return 0, err
	}
	return len(zones), nil
}

func ExtractCredentials(secret *unstructured.Unstructured) (string, string, error) {
	data, found, err := unstructured.NestedStringMap(secret.Object, "data")
	if err != nil {
		return "", "", fmt.Errorf("unable to extract data from secret: %w", err)
	}
	if !found {
		return "", "", fmt.Errorf("secret does not contain data")
	}

	accessKeyID, ok := data["accessKeyID"]
	if !ok {
		return "", "", fmt.Errorf("secret does not contain accessKeyID")
	}
	secretAccessKey, ok := data["secretAccessKey"]
	if !ok {
		return "", "", fmt.Errorf("secret does not contain secretAccessKey")
	}

	accessKeyIDBytes, err := base64.StdEncoding.DecodeString(accessKeyID)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode accessKeyID: %w", err)
	}
	secretAccessKeyBytes, err := base64.StdEncoding.DecodeString(secretAccessKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode secretAccessKey: %w", err)
	}

	return string(accessKeyIDBytes), string(secretAccessKeyBytes), nil
}

func newAWSConfig(ctx context.Context, key, secret, region string) (aws.Config, error) {
	return config.LoadDefaultConfig(
		ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(key, secret, "")),
		config.WithRegion(region),
	)
}
