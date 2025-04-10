package client

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

// MockIAMClient is a simple mock for the IAM client.
type MockIAMClient struct {
	// This is the AWS SDK function that we want to mock.
	// We can pass a function to this field to simulate different behaviors.
	GetRoleFunc func(input *iam.GetRoleInput) (*iam.GetRoleOutput, error)
}

func (m *MockIAMClient) GetRole(input *iam.GetRoleInput) (*iam.GetRoleOutput, error) {
	return m.GetRoleFunc(input)
}

func TestExtractServiceInfo(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedName string
		expectedEnv  string
		expectError  bool
	}{
		{
			name:         "Success - Valid service name",
			input:        "staging-my-service-service",
			expectedName: "my-service",
			expectedEnv:  "staging",
			expectError:  false,
		},
		{
			name:        "Failure - Invalid service name",
			input:       "invalid-service-format",
			expectError: true,
		},
		{
			name:        "Failure - Empty input",
			input:       "",
			expectError: true,
		},
		{
			name:        "Failure - Missing environment",
			input:       "my-service-service",
			expectError: true,
		},
		{
			name:        "Failure - Missing service identifier",
			input:       "staging--service",
			expectError: true,
		},
		{
			name:        "Failure - Invalid characters",
			input:       "staging-my_service-service",
			expectError: true,
		},
		{
			name:        "Failure - Extra suffix",
			input:       "staging-my-service-service-extra",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, env, err := extractServiceInfo(tc.input)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if name != tc.expectedName {
					t.Errorf("Expected name %q, but got %q", tc.expectedName, name)
				}
				if env != tc.expectedEnv {
					t.Errorf("Expected env %q, but got %q", tc.expectedEnv, env)
				}
			}
		})
	}
}

func TestGetRoleArn(t *testing.T) {
	tests := []struct {
		name        string
		roleName    string
		mockFunc    func(input *iam.GetRoleInput) (*iam.GetRoleOutput, error)
		expectedARN string
		expectError bool
	}{
		{
			name:     "Success - valid ARN",
			roleName: "my-service.production",
			mockFunc: func(input *iam.GetRoleInput) (*iam.GetRoleOutput, error) {
				roleArn := "arn:aws:iam::123456789012:role/internal/service/ecs-task/my-service.production"
				return &iam.GetRoleOutput{
					Role: &iam.Role{
						Arn: aws.String(roleArn),
					},
				}, nil
			},
			expectedARN: "arn:aws:iam::123456789012:role/internal/service/ecs-task/my-service.production",
			expectError: false,
		},
		{
			name:     "Failure - role not found",
			roleName: "nonexistent-role",
			mockFunc: func(input *iam.GetRoleInput) (*iam.GetRoleOutput, error) {
				return nil, fmt.Errorf("NoSuchEntity")
			},
			expectError: true,
		},
		{
			name:     "Failure - nil Role ARN",
			roleName: "my-service.production",
			mockFunc: func(input *iam.GetRoleInput) (*iam.GetRoleOutput, error) {
				return &iam.GetRoleOutput{
					Role: &iam.Role{
						Arn: nil,
					},
				}, nil
			},
			expectError: true,
		},
		{
			name:     "Failure - invalid ARN pattern",
			roleName: "my-service.production",
			mockFunc: func(input *iam.GetRoleInput) (*iam.GetRoleOutput, error) {
				roleArn := "non-conforming-arn"
				return &iam.GetRoleOutput{
					Role: &iam.Role{
						Arn: aws.String(roleArn),
					},
				}, nil
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var mockIAM iamInterface = &MockIAMClient{
				GetRoleFunc: tc.mockFunc,
			}
			client := &Client{
				iamClient: mockIAM,
			}
			arn, err := client.getRoleArn(tc.roleName)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if arn != tc.expectedARN {
					t.Errorf("Expected ARN %q, but got %q", tc.expectedARN, arn)
				}
			}
		})
	}
}

func TestGetTaskRole(t *testing.T) {
	tests := []struct {
		name           string
		currentRoleArn string
		taskRoleArn    string
		service        string
		mockFunc       func(input *iam.GetRoleInput) (*iam.GetRoleOutput, error)
		expectedARN    string
		expectError    bool
	}{
		{
			name:           "Success - Already in new format",
			currentRoleArn: "arn:aws:iam::123456789012:role/internal/service/ecs-task/my-service.production",
			service:        "staging-my-service-service",
			mockFunc:       nil, // No mock needed as function should return prior to reaching func.
			expectedARN:    "arn:aws:iam::123456789012:role/internal/service/ecs-task/my-service.production",
			expectError:    false,
		},
		{
			name:           "Success - Upgrade role format success",
			currentRoleArn: "arn:aws:iam::123456789012:role/old-format",
			service:        "staging-my-service-service",
			mockFunc: func(input *iam.GetRoleInput) (*iam.GetRoleOutput, error) {
				expectedRoleName := "my-service.staging"
				if input.RoleName != nil && *input.RoleName == expectedRoleName {
					roleArn := "arn:aws:iam::123456789012:role/internal/service/ecs-task/my-service.staging"
					return &iam.GetRoleOutput{
						Role: &iam.Role{
							Arn: aws.String(roleArn),
						},
					}, nil
				}
				return nil, fmt.Errorf("role not found")
			},
			expectedARN: "arn:aws:iam::123456789012:role/internal/service/ecs-task/my-service.staging",
			expectError: false,
		},
		{
			name:           "Success - TaskRoleArn is set and valid",
			currentRoleArn: "arn:aws:iam::123456789012:role/old-format",
			service:        "staging-my-service-service",
			mockFunc:       nil,
			taskRoleArn:    "arn:aws:iam::123456789012:role/test/some-other-role",
			expectedARN:    "arn:aws:iam::123456789012:role/test/some-other-role",
			expectError:    false,
		},
		{
			name:           "Failure - Upgrade role format failure",
			currentRoleArn: "arn:aws:iam::123456789012:role/old-format",
			service:        "testing-edgar-test-service",
			mockFunc: func(input *iam.GetRoleInput) (*iam.GetRoleOutput, error) {
				return nil, fmt.Errorf("role not found")
			},
			expectError: true,
		},
		{
			name:           "Failure - Nil currentRoleArn",
			currentRoleArn: "",
			service:        "staging-my-service-service",
			mockFunc:       nil,
			expectError:    true,
		},
		{
			name:           "Failure - Nil service",
			currentRoleArn: "arn:aws:iam::123456789012:role/old-format",
			service:        "",
			mockFunc:       nil,
			expectError:    true,
		},
		{
			name:           "Failure - extractServiceInfo failure",
			currentRoleArn: "arn:aws:iam::123456789012:role/old-format",
			service:        "invalid-service-name",
			mockFunc:       nil,
			expectError:    true,
		},
		{
			name:           "Failure - getRoleArn failure",
			currentRoleArn: "arn:aws:iam::123456789012:role/old-format",
			service:        "staging-my-service-service",
			mockFunc: func(input *iam.GetRoleInput) (*iam.GetRoleOutput, error) {
				return nil, fmt.Errorf("getRoleArn failed")
			},
			expectError: true,
		},
		{
			name:           "Failure - TaskRoleArn is set but is invalid",
			currentRoleArn: "arn:aws:iam::123456789012:role/old-format",
			service:        "staging-my-service-service",
			mockFunc:       nil,
			taskRoleArn:    "invalid-arn",
			expectError:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var mockIAM iamInterface
			if tc.mockFunc != nil {
				mockIAM = &MockIAMClient{
					GetRoleFunc: tc.mockFunc,
				}
			} else {
				// Dummy mock; should not be called in this case.
				mockIAM = &MockIAMClient{
					GetRoleFunc: func(input *iam.GetRoleInput) (*iam.GetRoleOutput, error) {
						return nil, fmt.Errorf("should not be called")
					},
				}
			}

			prefix := fmt.Sprintf("%s/%s ", "test-cluster", "test-service")
			logger := log.New(os.Stderr, prefix, log.LstdFlags)
			client := &Client{
				iamClient:   mockIAM,
				logger:      logger,
				taskRoleArn: tc.taskRoleArn,
			}
			result, err := client.getTaskRole(&tc.currentRoleArn, &tc.service)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Errorf("Expected result but got nil")
				} else if *result != tc.expectedARN {
					t.Errorf("Expected ARN %q, but got %q", tc.expectedARN, *result)
				}
			}
		})
	}
}
