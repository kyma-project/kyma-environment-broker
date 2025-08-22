package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Client struct {
	awsClient *ec2.Client
}

func NewClient(ctx context.Context, key, secret, region string) (*Client, error) {
	cfg, err := newAWSConfig(ctx, key, secret, region)
	if err != nil {
		return nil, fmt.Errorf("while creating AWS config: %w", err)
	}
	return &Client{awsClient: ec2.NewFromConfig(cfg)}, nil
}

func (c *Client) AvailableZones(ctx context.Context, machineType string) ([]string, error) {
	params := &ec2.DescribeInstanceTypeOfferingsInput{
		LocationType: "availability-zone",
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-type"),
				Values: []string{machineType},
			},
		},
	}
	resp, err := c.awsClient.DescribeInstanceTypeOfferings(ctx, params)
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

func newAWSConfig(ctx context.Context, key, secret, region string) (aws.Config, error) {
	return config.LoadDefaultConfig(
		ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(key, secret, "")),
		config.WithRegion(region),
	)
}
