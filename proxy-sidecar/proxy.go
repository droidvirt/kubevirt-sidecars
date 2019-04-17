package main

import (
	"bufio"
	"os/exec"
	"strconv"
	"time"

	"kubevirt.io/kubevirt/pkg/log"
)

const (
	socksLocalAddress  = "127.0.0.1"
	socksLoadlPort     = 1080
	socksEncryptMethod = "rc4-md5"
)

var socksLogger = log.Logger("socks")

func startSocksProxy(server string, port int, pwd string, stopChan chan struct{}) {
	// Spawn socks from monitor process in order to ensure the socks
	// process doesn't exit until monitor is ready for it to.
	// Monitor traps signals to perform special shutdown logic.
	// These processes need to live in the same container.

	go func() {
		for {
			exitChan := make(chan struct{})
			cmd := exec.Command("/usr/bin/ss-redir",
				"-s", server,
				"-p", strconv.Itoa(port),
				"-b", socksLocalAddress,
				"-l", strconv.Itoa(socksLoadlPort),
				"-k", pwd,
				"-m", socksEncryptMethod,
			)

			// connect socks's stderr to our own stdout in order to see the logs in the container logs
			stderrReader, err := cmd.StderrPipe()
			if err != nil {
				socksLogger.Reason(err).Error("failed to start socks")
				panic(err)
			}
			go func() {
				scanner := bufio.NewScanner(stderrReader)
				for scanner.Scan() {
					socksLogger.Errorf("socks error: %s", scanner.Text())
				}
				if err := scanner.Err(); err != nil {
					socksLogger.Reason(err).Error("failed to read socks stderr")
				}
			}()

			stdoutReader, err := cmd.StdoutPipe()
			if err != nil {
				socksLogger.Reason(err).Error("failed to start socks")
				panic(err)
			}
			go func() {
				scanner := bufio.NewScanner(stdoutReader)
				for scanner.Scan() {
					socksLogger.Infof("socks: %s", scanner.Text())
				}
				if err := scanner.Err(); err != nil {
					socksLogger.Reason(err).Error("failed to read socks stdout")
				}
			}()

			err = cmd.Start()
			if err != nil {
				socksLogger.Reason(err).Error("failed to start socks")
				panic(err)
			}

			go func() {
				defer close(exitChan)
				cmd.Wait()
			}()

			select {
			case <-stopChan:
				cmd.Process.Kill()
				return
			case <-exitChan:
				socksLogger.Errorf("socks exited, restarting")
			}

			// this sleep is to avoid consumming all resources in the
			// event of a libvirtd crash loop.
			time.Sleep(time.Second)
		}
	}()

}
