package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

var (
	buildVersion = "development"
	buildSha     = "devel"
)

const (
	maxWaitTimeRunning = 10 * time.Minute
)

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

func runTask(ecsClient *ecs.Client, taskDefArn string, containerDetails ContainerDetails, config RemoteShellOptions) string {
	log.Println("Launching console task...")

	var assignPublicIp types.AssignPublicIp = types.AssignPublicIpDisabled
	if containerDetails.config.AssignPublicIp {
		assignPublicIp = types.AssignPublicIpEnabled
	}

	var command []string = []string{
		"/cloud87/bin/remote-shell",
		"-user",
		config.githubUsername,
		"-port",
		strconv.Itoa(int(containerDetails.port)),
	}

	containerOverride := &types.ContainerOverride{
		Name:    aws.String(containerDetails.name),
		Command: command,
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

func main() {

	globalOptions := parseCommandFlags()

	log.Println("Starting Cloud87 Remote Shell Client")
	log.Printf("Version: %s@%s\n", buildVersion, buildSha)

	log.Println("Application:", globalOptions.applicationName)

	var awsConfigOptions []func(*config.LoadOptions) error

	if globalOptions.awsProfile != "" {
		awsConfigOptions = append(awsConfigOptions, config.WithSharedConfigProfile(globalOptions.awsProfile))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), awsConfigOptions...)
	if err != nil {
		panic(fmt.Sprintf("failed loading config, %v", err))
	}

	ecsClient := ecs.NewFromConfig(cfg)

	taskDefResult, err := ecsClient.DescribeTaskDefinition(context.TODO(), &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(globalOptions.applicationName),
	})
	check(err)

	taskDef := taskDefResult.TaskDefinition

	fmt.Println(aws.ToString(taskDef.TaskDefinitionArn))

	containerDetails := extractContainerDetails(taskDef)

	log.Println("Using Port:", containerDetails.port)

	if conf := askForConfirmation("Are you sure you want to launch a remote shell?"); !conf {
		log.Println("Alrighty then")
		os.Exit(0)
	}

	// Launch the task
	taskArn := runTask(ecsClient, aws.ToString(taskDef.TaskDefinitionArn), containerDetails, globalOptions)

	// should anything bad happen, kill it
	defer terminateTask(ecsClient, taskArn, containerDetails)

	log.Println("Waiting for task to successfully launch...")
	waiter := ecs.NewTasksRunningWaiter(ecsClient)
	var taskArns []string
	taskArns = append(taskArns, taskArn)

	params := &ecs.DescribeTasksInput{
		Cluster: aws.String(containerDetails.config.Cluster),
		Tasks:   taskArns,
	}

	err = waiter.Wait(context.TODO(), params, maxWaitTimeRunning, func(trwo *ecs.TasksRunningWaiterOptions) {
		trwo.MaxDelay = 15 * time.Second
	})
	if err != nil {
		log.Println(err)
	}

	taskInfoResult, err := ecsClient.DescribeTasks(context.TODO(), params)
	check(err)
	taskInfo := taskInfoResult.Tasks[0]

	log.Println(taskInfo)

	var containerAddress string
	if containerDetails.config.AssignPublicIp {
		ec2Client := ec2.NewFromConfig(cfg)

		containerAddress = extractContainerPublicAddress(ec2Client, taskInfo, containerDetails)
	} else {
		containerAddress = extractContainerPrivateAddress(taskInfo, containerDetails)
	}

	log.Println("Container IP:", containerAddress, containerDetails.port)

	launchSSHSession(globalOptions, containerAddress, containerDetails.port)

}
