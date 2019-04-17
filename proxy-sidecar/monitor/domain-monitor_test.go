package monitor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWaitLibvirtReady(t *testing.T) {
	mockReadinessDir := "/tmp/droidvirt/"
	mockReadinessFile := filepath.Join(mockReadinessDir, "healthy")

	os.MkdirAll(mockReadinessDir, 0755)
	defer os.RemoveAll(mockReadinessDir)

	go func() {
		time.Sleep(5 * time.Second)
		_, err := os.Create(mockReadinessFile)
		if err != nil {
			t.Errorf("Create mock sock file err: %s", err)
		}
	}()

	isReady, err := WaitLauncherReady(mockReadinessFile, 3)
	if err != nil || !isReady {
		t.Errorf("Wait libvirt ready error: %s", err)
	}
}
