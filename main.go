package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

var (
	buildVersion = "development"
	buildSha     = "devel"
)

const (
	maxWaitTimeRunning = 10 * time.Minute
)

var globalOptions RemoteShellOptions

func init() {
	globalOptions = parseCommandFlags()
}

func main() {

	defer func() {
		exitCode := 0
		if r := recover(); r != nil {
			exitCode = 1
			log.Println("FATAL ERROR! ", r)
		}
		os.Exit(exitCode)
	}()

	log.Println("Starting Cloud87 Remote Shell Client")
	log.Printf("Version: %s@%s\n", buildVersion, buildSha)

	log.Println("Application:", globalOptions.applicationName)

	awsCfg := loadAwsConfig(globalOptions)

	ecsClient := ecs.NewFromConfig(awsCfg)

	taskDefResult, err := ecsClient.DescribeTaskDefinition(context.TODO(), &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(globalOptions.applicationName),
	})
	if err != nil {
		// log.Fatalln(err.Error())
		panic(err)
	}

	taskDef := taskDefResult.TaskDefinition

	log.Println("Task Definition:", aws.ToString(taskDef.TaskDefinitionArn))

	containerDetails := extractContainerDetails(taskDef)

	log.Println("Using Port:", containerDetails.port)

	if globalOptions.interactive {
		if conf := askForConfirmation("Are you sure you want to launch a remote shell?"); !conf {
			log.Println("Alrighty then")
			os.Exit(0)
		}
	}

	// Launch the task
	taskArn := runTask(ecsClient, aws.ToString(taskDef.TaskDefinitionArn), containerDetails, globalOptions)

	// should anything bad happen, kill it
	defer terminateTask(ecsClient, taskArn, containerDetails)

	log.Println("Task ARN:", taskArn)
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

	// check to make sure the task is viable to connect to
	ensureHealthyTask(taskInfo, containerDetails)

	var containerAddress string
	if containerDetails.config.AssignPublicIp {
		ec2Client := ec2.NewFromConfig(awsCfg)

		containerAddress = extractContainerPublicAddress(ec2Client, taskInfo, containerDetails)
	} else {
		containerAddress = extractContainerPrivateAddress(taskInfo, containerDetails)
	}

	log.Println("Container IP:", containerAddress, containerDetails.port)

	launchSSHSession(globalOptions, containerAddress, containerDetails.port)

}
