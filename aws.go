package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

const sshAuthKeyEnvVar = "C87_RSHELL_AUTHORIZED_KEY"

func loadAwsConfig(options RemoteShellOptions) aws.Config {
	var awsConfigOptions []func(*config.LoadOptions) error

	if options.awsProfile != "" {
		awsConfigOptions = append(awsConfigOptions, config.WithSharedConfigProfile(options.awsProfile))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), awsConfigOptions...)
	if err != nil {
		panic(fmt.Sprintf("failed loading config, %v", err))
	}

	return cfg
}

func terminateTask(ecsClient *ecs.Client, taskArn string, containerDetails ContainerDetails) {
	log.Println("Terminating task", taskArn, "...")

	_, err := ecsClient.StopTask(context.TODO(), &ecs.StopTaskInput{
		Cluster: aws.String(containerDetails.config.Cluster),
		Task:    aws.String(taskArn),
		Reason:  aws.String("finished with remote shell"),
	})
	if err != nil {
		log.Printf("Failed to terminate task: %v\n", err)
	}

}

func ensureHealthyTask(taskInfo types.Task, containerDetails ContainerDetails) {

	for _, contInfo := range taskInfo.Containers {
		if contInfo.Reason != nil {
			log.Println("Container Stopped!")
			log.Println(aws.ToString(contInfo.Reason))
			panic("Container Stopped")
		}
	}

	if taskInfo.StopCode != "" {
		log.Println("Container Stopped!")
		log.Println(aws.ToString(taskInfo.StoppedReason))
		panic("Container Stopped")
	}

	desiredStatus := aws.ToString(taskInfo.DesiredStatus)
	if desiredStatus == "STOPPED" {
		panic("Container Stopped")
	}
}

func runTask(ecsClient *ecs.Client, taskDefArn string, containerDetails ContainerDetails, config RemoteShellOptions) string {
	log.Println("Launching console task...")

	var assignPublicIp types.AssignPublicIp = types.AssignPublicIpDisabled
	if containerDetails.config.AssignPublicIp {
		assignPublicIp = types.AssignPublicIpEnabled
	}

	commandPath := "/cloud87/remote-shell"
	if containerDetails.config.Path != "" {
		commandPath = containerDetails.config.Path
	}

	var command []string = []string{
		commandPath,
		"-port",
		strconv.Itoa(int(containerDetails.port)),
		"-maxtime",
		config.maxTime.String(),
		"-idletime",
		config.idleTime.String(),
	}

	var envVars = []types.KeyValuePair{
		{
			Name:  aws.String(sshAuthKeyEnvVar),
			Value: aws.String(config.authSshKey),
		},
	}

	containerOverride := &types.ContainerOverride{
		Name:        aws.String(containerDetails.name),
		Command:     command,
		Environment: envVars,
	}
	var containerOverrides = []types.ContainerOverride{*containerOverride}

	result, err := ecsClient.RunTask(context.TODO(), &ecs.RunTaskInput{
		TaskDefinition:  aws.String(taskDefArn),
		Cluster:         aws.String(containerDetails.config.Cluster),
		Count:           aws.Int32(1),
		StartedBy:       aws.String("cloud87/remoteshell-client"),
		LaunchType:      types.LaunchTypeFargate,
		PlatformVersion: aws.String("LATEST"),

		Overrides: &types.TaskOverride{
			ContainerOverrides: containerOverrides,
		},
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				AssignPublicIp: assignPublicIp,
				Subnets:        containerDetails.config.SubnetIds,
				SecurityGroups: containerDetails.config.SecurityGroupIds,
			},
		},
	})
	check(err)

	if len(result.Failures) > 0 {
		log.Panicf("Error Launching task!: %s: %s\n",
			aws.ToString(result.Failures[0].Detail),
			aws.ToString(result.Failures[0].Reason),
		)
		panic("unable to launch")
	}

	return aws.ToString(result.Tasks[0].TaskArn)
}
