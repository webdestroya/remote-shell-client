package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

type RemoteShellOptions struct {
	applicationName string
	githubUsername  string
	maxTime         time.Duration
	idleTime        time.Duration
	awsProfile      string
	forceKeyFile    string
}

func parseCommandFlags() RemoteShellOptions {
	var help = flag.Bool("help", false, "Show help")
	var version = flag.Bool("version", false, "Show version")
	var applicationFlag string
	var githubFlag string
	var exactFlag bool
	var idleTimeFlag time.Duration
	var maxTimeFlag time.Duration
	var awsProfileFlag string
	var forceKeyFlag string

	flag.StringVar(&applicationFlag, "app", "", "App Name")
	flag.StringVar(&githubFlag, "github", fetchEnvValue("C87_RSHELL_GITHUB_USERNAME", ""), "Your github username")
	flag.StringVar(&awsProfileFlag, "profile", "", "Override AWS Profile")
	flag.StringVar(&forceKeyFlag, "key", "", "Use a specific SSH keyfile")

	flag.DurationVar(&idleTimeFlag, "idletime", 0*time.Second, "Idle timeout")
	flag.DurationVar(&maxTimeFlag, "maxtime", 12*time.Hour, "Max session duration")

	flag.BoolVar(&exactFlag, "exact", false, "Exact match app name")

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *version {
		fmt.Printf("%s@%s\n", buildVersion, buildSha)
		os.Exit(0)
	}

	if applicationFlag == "" {
		log.Fatal("You must provide an application name")
	}

	if githubFlag == "" {
		log.Fatal("You must provide your GitHub username")
	}

	if !exactFlag {
		applicationFlag = fmt.Sprintf("c87-%s-console", applicationFlag)
	}

	return RemoteShellOptions{
		applicationName: applicationFlag,
		maxTime:         maxTimeFlag,
		idleTime:        idleTimeFlag,
		awsProfile:      awsProfileFlag,
		githubUsername:  githubFlag,
		forceKeyFile:    forceKeyFlag,
	}
}
