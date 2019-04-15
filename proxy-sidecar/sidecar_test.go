package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWaitLibvirtReady(t *testing.T) {
	mockUID := "42"
	mockShareDir := "/tmp/libvirt"
	mockSocketsDir := filepath.Join(mockShareDir, "sockets")
	mockSockPath := filepath.Join(mockSocketsDir, mockUID+"_sock")

	os.MkdirAll(mockSocketsDir, 0755)
	defer os.RemoveAll(mockShareDir)

	go func() {
		time.Sleep(5 * time.Second)
		_, err := os.Create(mockSockPath)
		if err != nil {
			t.Errorf("Create mock sock file err: %s", err)
		}
	}()

	isReady, err := waitLibvirtReady(mockShareDir, mockUID, 3)
	if err != nil || !isReady {
		t.Errorf("Wait libvirt ready error: %s", err)
	}
}
