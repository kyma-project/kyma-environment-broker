package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

type mockEC2Client struct {
	describeFn func(ctx context.Context, params *ec2.DescribeInstanceTypeOfferingsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypeOfferingsOutput, error)
}

func (m *mockEC2Client) DescribeInstanceTypeOfferings(ctx context.Context, params *ec2.DescribeInstanceTypeOfferingsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypeOfferingsOutput, error) {
	return m.describeFn(ctx, params, optFns...)
}

func TestAvailableZones_Success(t *testing.T) {
	mock := &mockEC2Client{
		describeFn: func(ctx context.Context, params *ec2.DescribeInstanceTypeOfferingsInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstanceTypeOfferingsOutput, error) {
			return &ec2.DescribeInstanceTypeOfferingsOutput{
				InstanceTypeOfferings: []types.InstanceTypeOffering{
					{Location: aws.String("ap-southeast-2a")},
					{Location: aws.String("ap-southeast-2c")},
				},
			}, nil
		},
	}

	client := &AWSClient{ec2Client: mock}

	zones, err := client.AvailableZones(context.Background(), "g6.xlarge")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"ap-southeast-2a", "ap-southeast-2c"}, zones)
}

func TestAvailableZones_Error(t *testing.T) {
	mock := &mockEC2Client{
		describeFn: func(ctx context.Context, params *ec2.DescribeInstanceTypeOfferingsInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstanceTypeOfferingsOutput, error) {
			return nil, errors.New("AWS error")
		},
	}

	client := &AWSClient{ec2Client: mock}

	zones, err := client.AvailableZones(context.Background(), "g6.xlarge")
	assert.EqualError(t, err, "failed to describe offerings: AWS error")
	assert.Nil(t, zones)
}

func TestAvailableZones_NoLocations(t *testing.T) {
	mock := &mockEC2Client{
		describeFn: func(ctx context.Context, params *ec2.DescribeInstanceTypeOfferingsInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstanceTypeOfferingsOutput, error) {
			return &ec2.DescribeInstanceTypeOfferingsOutput{
				InstanceTypeOfferings: []types.InstanceTypeOffering{
					{Location: nil},
				},
			}, nil
		},
	}

	client := &AWSClient{ec2Client: mock}

	zones, err := client.AvailableZones(context.Background(), "g6.xlarge")
	assert.NoError(t, err)
	assert.Empty(t, zones)
}
