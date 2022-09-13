package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

const (
	maxKeyFileSize = 2 * 1024 * 1024 // 2 megabytes

	sshBaudSpeed = 14400 // 14400,
	// sshBaudSpeed = 38400 // 14400,
)

func loadKeyFile(filepath string) ssh.Signer {
	key, err := os.ReadFile(filepath)
	if err != nil {
		return nil
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil
	}

	return signer
}

func loadLocalKeys(options RemoteShellOptions) []ssh.Signer {

	if options.forceKeyFile != "" {
		return []ssh.Signer{loadKeyFile(options.forceKeyFile)}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("Unable to determine user home directory")
	}

	sshDir := fmt.Sprintf("%s/.ssh", homeDir)

	files, err := os.ReadDir(sshDir)
	if err != nil {
		panic("unable to read keys from ssh directory")
	}

	var signers []ssh.Signer

	for _, file := range files {
		if file.Type().IsRegular() {
			info, err := file.Info()
			if err != nil {
				continue
			}
			if info.Size() <= maxKeyFileSize {
				keyFile := fmt.Sprintf("%s/%s", sshDir, file.Name())
				signer := loadKeyFile(keyFile)
				if signer != nil {
					log.Println("Loaded key", keyFile)
					signers = append(signers, signer)

				}
			}
		}
	}

	return signers
}

func launchSSHSession(options RemoteShellOptions, containerHost string, port int32) {

	sshKeys := loadLocalKeys(options)

	if len(sshKeys) == 0 {
		log.Panicln("Could not load any keys!")
	}

	config := &ssh.ClientConfig{
		User:            "cloud87",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		ClientVersion:   fmt.Sprintf("SSH-2.0-cloud87-client-%s", buildVersion), // MUST start with SSH-2.0-
		Timeout:         30 * time.Second,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(sshKeys...),
		},
	}

	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGWINCH)

	containerAddress := fmt.Sprintf("%s:%d", containerHost, port)
	client, err := ssh.Dial("tcp", containerAddress, config)
	check(err)
	defer client.Close()

	session, err := client.NewSession()
	check(err)
	defer session.Close()

	termFd := int(os.Stdin.Fd())

	width, height, err := term.GetSize(termFd)
	if err != nil {
		log.Println("Failed to get terminal size")
		height = 24
		width = 80
	}
	log.Printf("Terminal Size: %d rows x %d cols\n", height, width)

	fmt.Println("")

	state, err := term.MakeRaw(termFd)
	if err != nil {
		panic(err)
	}
	defer term.Restore(termFd, state)

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: sshBaudSpeed,
		ssh.TTY_OP_OSPEED: sshBaudSpeed,
	}

	session.Stdin = os.Stdin
	session.Stderr = os.Stderr
	session.Stdout = os.Stdout

	go func() {
		for value := range signalChan {
			if value == syscall.SIGINT {
				session.Signal(ssh.SIGINT)

			} else if value == syscall.SIGTERM {
				session.Signal(ssh.SIGTERM)

			} else if value == syscall.SIGWINCH {
				width, height, err := term.GetSize(termFd)
				if err == nil {
					_ = session.WindowChange(height, width)
				}
			} else {
				// uhhh?
			}
		}
	}()

	termName := fetchEnvValue("TERM", "xterm")

	if err := session.RequestPty(termName, int(height), int(width), modes); err != nil {
		log.Panicln("Unable to request PTY session: ", err)
	}

	if err := session.Shell(); err != nil {
		log.Panicln("Unable to shell: ", err)
	}

	if err := session.Wait(); err != nil {
		panic(err)
	}
	close(signalChan)
}
