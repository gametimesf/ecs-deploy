package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gametimesf/ecs-deploy/client"
)

var (
	service     = flag.String("service", "", "Name of Service to update. Required.")
	image       = flag.String("image", "", "Name of Docker image to run.")
	tag         = flag.String("tag", "", "Tag of Docker image to run.")
	cluster     = flag.String("cluster", "default", "Name of ECS cluster.")
	task        = flag.String("task", "", "Name of task definition. Defaults to service name")
	region      = flag.String("region", "us-east-1", "Name of AWS region.")
	count       = flag.Int64("count", -1, "Desired count of instantiations to place and run in service. Defaults to existing running count.")
	nowait      = flag.Bool("nowait", false, "Disable waiting for all task definitions to start running")
	canary      = flag.Bool("canary", false, "Use canary deployment strategy")
	app         = flag.String("app", "", "CodeDeploy application name")
	deploygroup = flag.String("deploygroup", "", "CodeDeploy deployment group name")
	port        = flag.Int64("port", -1, "Service port")
)

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

	prefix := fmt.Sprintf("%s/%s ", *cluster, *service)
	logger := log.New(os.Stderr, prefix, log.LstdFlags)
	c := client.New(region, logger)

	arn := ""
	var err error

	if image != nil {
		arn, err = c.RegisterTaskDefinition(task, image, tag)
		if err != nil {
			logger.Printf("[error] register task definition: %s\n", err)
			os.Exit(1)
		}
	}

	if *canary {
		err = c.CreateDeployment(app, deploygroup, service, port, &arn)
		if err != nil {
			logger.Printf("[error] create deployment: %s\n", err)
			os.Exit(1)
		}
	} else {
		err = c.UpdateService(cluster, service, count, &arn)
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
	}

	logger.Printf("[info] update service success")
	os.Exit(0)
}
