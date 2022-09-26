# Fargate Remote Shell Launcher

This program makes it dead simple to launch a task using the [webdestroya/remote-shell](https://github.com/webdestroya/remote-shell) service.

## Installation


## Usage

To launch a remote shell session, you only need to know the task definition prefix. By default, it is assumed that your task definition is whatever you enter for the `app` parameter, but with `-console` appended.

If you do not have a console suffix, then add `-exact` to your command.

```
remote-shell-client -app myapp
```

## Task Configuration
To use this, you must add a docker label to your ECS Task Definition on the container that will be used for the shell.

The label must be named `cloud87.rshell` and should contain a JSON object with the following fields:

| Field | Type | Value |
| ----- | ---- | ---- |
| `cluster` | String | The name of the ECS cluster to run the task on |
| `subnets` | Array | List of SubnetIDs to use for the network interface |
| `security_groups` | Array | List of SecurityGroupIDs to use for the network interface |
| `port` | Integer | The port that should be used for the SSH service |
| `public` | Boolean | Whether or not this container will be given a public IP address |
| `path` | String | Path to the remote-shell binary. If not provided, then `/cloud87/remote-shell` is assumed. |




## AWS Permissions Required
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "ecs:DescribeTaskDefinition",
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ecs:StopTask",
        "ecs:RunTask",
        "ecs:DescribeTasks"
      ],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "ecs:cluster": [
            "arn:aws:ecs:region:1234567890:cluster/CLUSTER_NAME"
          ]
        }
      }
    },
  ]
}
```

If your task has a role attached to it, you will also need to grant permission to pass that:

```json
{
  "Sid": "RolePassingForECS",
  "Effect": "Allow",
  "Action": "iam:PassRole",
  "Resource": [
    "arn:aws:iam::1234567890:role/EXECUTION_ROLE_NAME",
    "arn:aws:iam::1234567890:role/TASK_ROLE_NAME",
  ],
  "Condition": {
    "StringEquals": {
      "iam:PassedToService": [
        "ecs-tasks.amazonaws.com"
      ]
    }
  }
}
```

If you are using public facing sessions, you will need the following:

```json
{
  "Sid": "QueryPublicIps",
  "Effect": "Allow",
  "Action": "ec2:DescribeNetworkInterfaces",
  "Resource": "*"
}
```