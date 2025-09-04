package aws

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

func TestExtractCredentials(t *testing.T) {
	testCases := []struct {
		name            string
		unstructured    *unstructured.Unstructured
		error           error
		accessKeyID     string
		secretAccessKey string
	}{
		{
			name: "no data",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			error: fmt.Errorf("secret does not contain data"),
		},
		{
			name: "no accessKeyID",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"data": map[string]interface{}{
						"secretAccessKey": "dGVzdC1zZWNyZXQ=",
					},
				},
			},
			error: fmt.Errorf("secret does not contain accessKeyID"),
		},
		{
			name: "no secretAccessKey",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"data": map[string]interface{}{
						"accessKeyID": "dGVzdC1rZXk=",
					},
				},
			},
			error: fmt.Errorf("secret does not contain secretAccessKey"),
		},
		{
			name: "invalid accessKeyID",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"data": map[string]interface{}{
						"accessKeyID":     "test-key",
						"secretAccessKey": "dGVzdC1zZWNyZXQ=",
					},
				},
			},
			error: fmt.Errorf("failed to decode accessKeyID"),
		},
		{
			name: "invalid secretAccessKey",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"data": map[string]interface{}{
						"accessKeyID":     "dGVzdC1rZXk=",
						"secretAccessKey": "test-secret",
					},
				},
			},
			error: fmt.Errorf("failed to decode secretAccessKey"),
		},
		{
			name: "valid credentials",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"data": map[string]interface{}{
						"accessKeyID":     "dGVzdC1rZXk=",
						"secretAccessKey": "dGVzdC1zZWNyZXQ=",
					},
				},
			},
			accessKeyID:     "test-key",
			secretAccessKey: "test-secret",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			accessKeyID, secretAccessKey, err := ExtractCredentials(tc.unstructured)

			// then
			if tc.error != nil {
				assert.Contains(t, err.Error(), tc.error.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.accessKeyID, accessKeyID)
				assert.Equal(t, tc.secretAccessKey, secretAccessKey)
			}
		})
	}
}
