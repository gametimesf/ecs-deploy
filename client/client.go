package client

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/iam"
)

// ecsInterface for the ECS client.
type ecsInterface interface {
	RegisterTaskDefinition(*ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error)
	UpdateService(*ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error)
	DescribeServices(*ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error)
	DescribeTaskDefinition(*ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error)
}

// iamInterface interface for IAM client
type iamInterface interface {
	GetRole(*iam.GetRoleInput) (*iam.GetRoleOutput, error)
}

type Client struct {
	ecsClient    ecsInterface
	iamClient    iamInterface
	logger       *log.Logger
	pollInterval time.Duration
	dryRun       bool
	taskRoleArn  string
}

// New creates a new Client.
func New(region *string, logger *log.Logger, dryRun bool, taskRoleArn string) *Client {
	session, err := session.NewSession(&aws.Config{Region: region})
	if err != nil {
		logger.Fatalf("[error] failed to create session: %s\n", err)
	}
	ecsClient := ecs.New(session)
	iamClient := iam.New(session)
	return &Client{
		ecsClient:    ecsClient,
		iamClient:    iamClient,
		pollInterval: time.Second * 5,
		logger:       logger,
		dryRun:       dryRun,
		taskRoleArn:  taskRoleArn,
	}
}

// RegisterTaskDefinition updates the existing task definition's image, upgrading the task role if possible.
func (c *Client) RegisterTaskDefinition(task, image, tag, service *string) (string, error) {
	taskDef, err := c.GetTaskDefinition(task)
	if err != nil {
		return "", err
	}

	defs := taskDef.ContainerDefinitions
	for _, d := range defs {
		if strings.HasPrefix(*d.Image, *image) {
			i := fmt.Sprintf("%s:%s", *image, *tag)
			d.Image = &i
		}
	}

	// Get the manually set task role or upgrade the task role if possible.
	// If any errors occur, use the existing task role.
	taskRoleArn, err := c.getTaskRole(taskDef.TaskRoleArn, service)
	if err != nil {
		c.logger.Printf("[warn] Error getting task role: %v", err)
	}

	input := &ecs.RegisterTaskDefinitionInput{
		Family:                  task,
		TaskRoleArn:             taskRoleArn,
		NetworkMode:             taskDef.NetworkMode,
		ContainerDefinitions:    defs,
		Volumes:                 taskDef.Volumes,
		PlacementConstraints:    taskDef.PlacementConstraints,
		RequiresCompatibilities: taskDef.RequiresCompatibilities,
		ExecutionRoleArn:        taskDef.ExecutionRoleArn,
		Cpu:                     taskDef.Cpu,
		Memory:                  taskDef.Memory,
	}

	if c.dryRun {
		c.logger.Printf("[dry-run] RegisterTaskDefinition input: %v\n", input)
		return "dry-run-taskdef-arn", nil
	}

	resp, err := c.ecsClient.RegisterTaskDefinition(input)
	if err != nil {
		return "", err
	}
	return *resp.TaskDefinition.TaskDefinitionArn, nil
}

// UpdateService updates the service to use the new task definition.
func (c *Client) UpdateService(cluster, service *string, count *int64, arn *string) error {
	input := &ecs.UpdateServiceInput{
		Cluster: cluster,
		Service: service,
	}
	if *count != -1 {
		input.DesiredCount = count
	}
	if arn != nil {
		input.TaskDefinition = arn
	}

	if c.dryRun {
		c.logger.Printf("[dry-run] UpdateService input: %v\n", input)
		return nil
	}

	_, err := c.ecsClient.UpdateService(input)
	return err
}

// Wait waits for the service to finish being updated.
func (c *Client) Wait(cluster, service, arn *string) error {
	t := time.NewTicker(c.pollInterval)
	for {
		select {
		case <-t.C:
			s, err := c.GetDeployment(cluster, service, arn)
			if err != nil {
				return err
			}
			c.logger.Printf("[info] --> desired: %d, pending: %d, running: %d", *s.DesiredCount, *s.PendingCount, *s.RunningCount)
			if *s.RunningCount == *s.DesiredCount {
				return nil
			}
		}
	}
}

// GetDeployment gets the deployment for the arn.
func (c *Client) GetDeployment(cluster, service, arn *string) (*ecs.Deployment, error) {
	ds, err := c.GetDeployments(cluster, service)
	if err != nil {
		return nil, err
	}
	for _, d := range ds {
		if *d.TaskDefinition == *arn {
			return d, nil
		}
	}
	return nil, nil
}

// GetDeployments gets the deployments for service in the cluster.
func (c *Client) GetDeployments(cluster, service *string) ([]*ecs.Deployment, error) {
	input := &ecs.DescribeServicesInput{
		Cluster:  cluster,
		Services: []*string{service},
	}
	output, err := c.ecsClient.DescribeServices(input)
	if err != nil {
		return nil, err
	}
	return output.Services[0].Deployments, nil
}

// GetTaskDefinition gets the latest revision for the given task definition
func (c *Client) GetTaskDefinition(task *string) (*ecs.TaskDefinition, error) {
	output, err := c.ecsClient.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: task,
	})
	if err != nil {
		return nil, err
	}
	return output.TaskDefinition, nil
}
