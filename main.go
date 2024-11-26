package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gametimesf/ecs-deploy/client"
)

var (
	service              = flag.String("service", "", "Name of Service to update. Required.")
	image                = flag.String("image", "", "Name of Docker image to run.")
	tag                  = flag.String("tag", "", "Tag of Docker image to run.")
	cluster              = flag.String("cluster", "default", "Name of ECS cluster.")
	task                 = flag.String("task", "", "Name of task definition. Defaults to service name")
	region               = flag.String("region", "us-east-1", "Name of AWS region.")
	count                = flag.Int64("count", -1, "Desired count of instantiations to place and run in service. Defaults to existing running count.")
	nowait               = flag.Bool("nowait", false, "Disable waiting for all task definitions to start running")
	requireLatest        = flag.Bool("require-latest", true, "Require the latest task definition to be running")
	enableECSManagedTags = flag.Bool("enable-ecs-managed-tags", false, "Enable ECS managed tags to be automatically added to running tasks.")
	propagateTags        = flag.String("propagate-tags", "NONE", "Propagate custom tags to the task. The value indicates the source of tags TASK_DEFINITION, SERVICE, or NONE.")
)

func isValid(value string, validValues []string) bool {
	for _, v := range validValues {
		if value == v {
			return true
		}
	}
	return false
}

func main() {
	flag.Parse()

	if *service == "" {
		_, _ = fmt.Fprintln(os.Stderr, "service name is required")
		flag.Usage()
		os.Exit(1)
	}

	if *task == "" {
		task = service
	}

	if *region == "" {
		r := os.Getenv("AWS_DEFAULT_REGION")
		region = &r
	}

	validPropagateTags := []string{"NONE", "TASK_DEFINITION", "SERVICE"}
	if !isValid(*propagateTags, validPropagateTags) {
		_, _ = fmt.Fprintln(os.Stderr, "propagate-tags must be one of NONE, TASK_DEFINITION, or SERVICE")
		flag.Usage()
		os.Exit(1)
	}

	prefix := fmt.Sprintf("%s/%s ", *cluster, *service)
	logger := log.New(os.Stderr, prefix, log.LstdFlags)
	c := client.New(region, logger)

	arn := ""
	var err error

	if *requireLatest {
		// check if the latest task definition is running. if it is not, do not proceed with the deployment.
		taskDef, err := c.GetTaskDefinition(task)
		if err != nil {
			logger.Printf("[error] get task definition: %s\n", err)
			os.Exit(1)
		}

		deployments, err := c.GetDeployments(cluster, service)
		// there can be more than 1 deployment running if a deployment is in progress,
		// not worth handling at this point: it's not possible to determine if the latest deployment
		// will succeed, for example.
		if len(deployments) != 1 {
			logger.Printf("[error] not exactly one deployment found: %d\n", len(deployments))
			os.Exit(1)
		}

		d := deployments[0]
		if *d.TaskDefinition != *taskDef.TaskDefinitionArn {
			logger.Printf("[error] latest task definition not running: %s\n", *d.TaskDefinition)
			os.Exit(1)
		}

		logger.Printf("[info] latest task definition running: %s\n", *d.TaskDefinition)
	}

	if image != nil {
		arn, err = c.RegisterTaskDefinition(task, image, tag)
		if err != nil {
			logger.Printf("[error] register task definition: %s\n", err)
			os.Exit(1)
		}
	}

	err = c.UpdateService(cluster, service, count, &arn, enableECSManagedTags, propagateTags)
	if err != nil {
		logger.Printf("[error] update service: %s\n", err)
		os.Exit(1)
	}

	if *nowait == false {
		err := c.Wait(cluster, service, &arn)
		if err != nil {
			logger.Printf("[error] wait: %s\n", err)
			os.Exit(1)
		}
	}

	logger.Printf("[info] update service success")
	os.Exit(0)
}
