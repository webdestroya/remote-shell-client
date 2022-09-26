package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type RemoteShellDockerLabel struct {
	Cluster          string   `json:"cluster"`
	SubnetIds        []string `json:"subnets"`
	SecurityGroupIds []string `json:"security_groups"`
	AssignPublicIp   bool     `json:"public"`
	Port             int32    `json:"port"`
	Path             string   `json:"path,omitempty"`
}

type ContainerDetails struct {
	name         string
	port         int32
	containerDef ecsTypes.ContainerDefinition
	config       RemoteShellDockerLabel
}

func extractContainerDetails(taskDef *ecsTypes.TaskDefinition) ContainerDetails {
	var rshellConfig RemoteShellDockerLabel
	var containerName string
	var containerDef ecsTypes.ContainerDefinition
	// var containerAddress string

	for _, value := range taskDef.ContainerDefinitions {
		// check if container has our label
		if rshellConfigJson, ok := value.DockerLabels["cloud87.rshell"]; ok {

			// parse the json of the label
			if err := json.NewDecoder(strings.NewReader(rshellConfigJson)).Decode(&rshellConfig); err == nil {
				// fmt.Println("OK")
				// fmt.Println(rshellConfig)
				containerName = aws.ToString(value.Name)
				containerDef = value
				break
			}
		}
	}

	if containerName == "" {
		panic("This task definition does not have Cloud87 Remote Shell (cloud87.rshell) configuration label!")
	}

	return ContainerDetails{
		name:         containerName,
		port:         extractContainerPort(containerDef),
		containerDef: containerDef,
		config:       rshellConfig,
	}
}

func extractContainerPort(containerDef ecsTypes.ContainerDefinition) int32 {

	var desiredPorts = []int32{8722, 22}

	if len(containerDef.PortMappings) == 0 {
		panic("There are no open ports on this task definition!")
	}

	// search in order of preference for possible remote shell ports
	for _, desiredPort := range desiredPorts {
		for _, value := range containerDef.PortMappings {
			if value.Protocol == ecsTypes.TransportProtocolTcp && aws.ToInt32(value.ContainerPort) == desiredPort {
				return desiredPort
			}
		}
	}

	// nothing was found...
	log.Println("Could not determine port for remote shell... falling back to any open port.")
	for _, value := range containerDef.PortMappings {
		if value.Protocol == ecsTypes.TransportProtocolTcp {
			return aws.ToInt32(value.ContainerPort)
		}
	}

	panic("There are no open TCP ports on this task definition")
}

func extractContainerPrivateAddress(taskInfo ecsTypes.Task, containerDetails ContainerDetails) string {

	for _, value := range taskInfo.Containers {
		if aws.ToString(value.Name) == containerDetails.name {
			if len(value.NetworkInterfaces) > 0 {
				return aws.ToString(value.NetworkInterfaces[0].PrivateIpv4Address)
			}
			// value.NetworkInterfaces[0].PrivateIpv4Address
		}
	}

	panic("Container does not have any IPs?")
}

func extractContainerPublicAddress(ec2Client *ec2.Client, taskInfo ecsTypes.Task, containerDetails ContainerDetails) string {

	var attachmentId string

	for _, value := range taskInfo.Containers {
		if aws.ToString(value.Name) == containerDetails.name {
			if len(value.NetworkInterfaces) > 0 {
				attachmentId = aws.ToString(value.NetworkInterfaces[0].AttachmentId)
				break
			}
			// value.NetworkInterfaces[0].PrivateIpv4Address
		}
	}

	if attachmentId == "" {
		panic("Unable to find attached NetworkInterface")
	}

	var networkInferfaceId string
	for _, value := range taskInfo.Attachments {
		if aws.ToString(value.Id) == attachmentId {
			for _, detailValue := range value.Details {
				if aws.ToString(detailValue.Name) == "networkInterfaceId" {
					networkInferfaceId = aws.ToString(detailValue.Value)
					break
				}
			}
		}
	}

	if networkInferfaceId == "" {
		panic("Unable to determine NetworkInterfaceID for attachment")
	}

	result, err := ec2Client.DescribeNetworkInterfaces(context.TODO(), &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []string{networkInferfaceId},
	})
	check(err)

	networkInterface := result.NetworkInterfaces[0]

	if networkInterface.Association == nil {
		panic("No association information")
	}

	publicIp := aws.ToString(networkInterface.Association.PublicIp)

	if publicIp == "" {
		panic("No public IP for this ENI")
	}

	return publicIp
}
