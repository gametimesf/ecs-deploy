package client

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

var (
	taskRoleRegex    = regexp.MustCompile(`arn:aws:iam::\d{12}:role/internal/service/ecs-task/[a-z-]+\.(testing|staging|production)$`)
	serviceNameRegex = regexp.MustCompile(`(testing|staging|production)-([a-z-]+)-service$`)
)

// getTaskRole attempts to determine the correct task role ARN based on the service name
// and checks if it exists in AWS
func (c *Client) getTaskRole(currentRoleArn, service *string) (*string, error) {
	// If a task role ARN is provided, return that
	if c.taskRoleArn != "" {
		return &c.taskRoleArn, nil
	}

	// Input validation
	if currentRoleArn == nil || *currentRoleArn == "" {
		return currentRoleArn, fmt.Errorf("current role ARN is nil or empty")
	}

	if service == nil || *service == "" {
		return currentRoleArn, fmt.Errorf("service is nil or empty")
	}

	// If already using the new format, return it
	if taskRoleRegex.MatchString(*currentRoleArn) {
		return currentRoleArn, nil
	}

	// Extract service name and environment
	serviceName, env, err := extractServiceInfo(*service)
	if err != nil {
		return currentRoleArn, err
	}

	// Get the role name and verify it exists
	newRoleName := serviceName + "." + env
	newRoleArn, err := c.getRoleArn(newRoleName)
	if err != nil {
		return currentRoleArn, err
	}

	c.logger.Printf("[info] Upgrading task role:\n - Before: %s\n - After: %s", *currentRoleArn, newRoleArn)
	return &newRoleArn, nil
}

// extractServiceInfo extracts the service name and environment from a service identifier
func extractServiceInfo(service string) (string, string, error) {
	if match := serviceNameRegex.MatchString(service); !match {
		return "", "", fmt.Errorf("service name does not match expected format")
	}

	service = strings.TrimSuffix(service, "-service")
	env := strings.Split(service, "-")[0]
	name := strings.TrimPrefix(service, env+"-")

	if env == "" || name == "" {
		return "", "", fmt.Errorf("env or name is empty after parsing")
	}

	return name, env, nil
}

// getRoleArn verifies that the role exists and returns its ARN if it matches the expected pattern
func (c *Client) getRoleArn(roleName string) (string, error) {
	roleData, err := c.iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})

	if err != nil {
		return "", err
	}

	// Probably not needed, but don't want a nil pointer dereference in next check
	if roleData.Role.Arn == nil {
		return "", fmt.Errorf("role ARN is nil")
	}
	if !taskRoleRegex.MatchString(*roleData.Role.Arn) {
		return "", fmt.Errorf("role ARN %s doesn't conform to expected pattern", aws.StringValue(roleData.Role.Arn))
	}

	return *roleData.Role.Arn, nil
}
