package main

import (
	"bufio"
	"os/exec"
	"time"

	"kubevirt.io/kubevirt/pkg/log"
)

func startSocksProxy(stopChan chan struct{}) {
	// Spawn socks from monitor process in order to ensure the socks
	// process doesn't exit until monitor is ready for it to.
	// Monitor traps signals to perform special shutdown logic.
	// These processes need to live in the same container.

	go func() {
		for {
			exitChan := make(chan struct{})
			cmd := exec.Command("/usr/bin/ss-redir")

			// connect socks's stderr to our own stdout in order to see the logs in the container logs
			reader, err := cmd.StderrPipe()
			if err != nil {
				log.Log.Reason(err).Error("failed to start socks")
				panic(err)
			}

			go func() {
				scanner := bufio.NewScanner(reader)
				for scanner.Scan() {
					log.Log.Errorf("socks error: %s", scanner.Text())
				}

				if err := scanner.Err(); err != nil {
					log.Log.Reason(err).Error("failed to read socks logs")
				}
			}()

			err = cmd.Start()
			if err != nil {
				log.Log.Reason(err).Error("failed to start socks")
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
				log.Log.Errorf("socks exited, restarting")
			}

			// this sleep is to avoid consumming all resources in the
			// event of a libvirtd crash loop.
			time.Sleep(time.Second)
		}
	}()

}
