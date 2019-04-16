package iptables

import (
	"github.com/golang/mock/gomock"
	"testing"
)

var (
	mockIptablesRules = []string{
		"-A KUBEVIRT_POSTINBOUND -p tcp -m tcp --dport 22 -j SNAT --to-source 192.168.100.1",
	}
)

func TestParseForwardPorts(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockHandler := NewMockIptablesHandler(mockCtrl)

	mockHandler.EXPECT().iptablesListRules(rulesTable, kubevirtForwardChain).Return(mockIptablesRules, nil).Times(1)

	ports, err := ParseForwardPorts(mockHandler)

	if err != nil {
		t.Errorf("Parse ports error: %s", err)
	}

	t.Logf("ports: %v", ports)
	if len(ports) != 1 || ports[0] != 22 {
		t.Fail()
	}
}
