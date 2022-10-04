package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

type RemoteShellOptions struct {
	applicationName string
	maxTime         time.Duration
	idleTime        time.Duration
	awsProfile      string
	checkMode       bool
	sshKeyBits      int
	privateKey      ssh.Signer
	authSshKey      string
	interactive     bool

	// githubUsername  string
	// forceKeyFile    string
}

func parseCommandFlags() RemoteShellOptions {
	var help = flag.Bool("help", false, "Show help")
	var version = flag.Bool("version", false, "Show version")
	var applicationFlag string
	var exactFlag bool
	var checkFlag bool = false
	var interactiveFlag bool = false
	var idleTimeFlag time.Duration
	var maxTimeFlag time.Duration
	var awsProfileFlag string
	var sshKeyBits int
	// var githubFlag string
	// var forceKeyFlag string

	flag.StringVar(&applicationFlag, "app", "", "App Name")
	// flag.StringVar(&githubFlag, "github", fetchEnvValue("C87_RSHELL_GITHUB_USERNAME", ""), "Your github username")
	flag.StringVar(&awsProfileFlag, "profile", "", "Override AWS Profile")
	// flag.StringVar(&forceKeyFlag, "key", "", "Use a specific SSH keyfile")

	flag.DurationVar(&idleTimeFlag, "idletime", 0*time.Second, "Idle timeout")
	flag.DurationVar(&maxTimeFlag, "maxtime", 12*time.Hour, "Max session duration")

	flag.IntVar(&sshKeyBits, "bits", 4096, "Key Bit Length")

	flag.BoolVar(&exactFlag, "exact", false, "Exact match app name")
	flag.BoolVar(&interactiveFlag, "interactive", false, "Ask for confirmation before launching")
	// flag.BoolVar(&checkFlag, "check", checkFlag, "Check Mode (Dry Run)")

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

	if !exactFlag {
		applicationFlag = fmt.Sprintf("%s-console", applicationFlag)
	}

	privateKey, authPubKey, err := generateSSHKeypair(sshKeyBits)
	if err != nil {
		panic(err)
	}

	return RemoteShellOptions{
		applicationName: applicationFlag,
		maxTime:         maxTimeFlag,
		idleTime:        idleTimeFlag,
		awsProfile:      awsProfileFlag,
		checkMode:       checkFlag,
		privateKey:      privateKey,
		authSshKey:      authPubKey,
		sshKeyBits:      sshKeyBits,
		interactive:     interactiveFlag,
	}
}
