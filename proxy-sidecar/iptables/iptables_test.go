package iptables

import (
	"github.com/golang/mock/gomock"
	"testing"
)

var (
	mockForwardRules = []string{
		"-A KUBEVIRT_POSTINBOUND -p tcp -m tcp --dport 22 -j SNAT --to-source 192.168.100.1",
		"-A KUBEVIRT_POSTINBOUND -p tcp -m tcp",
		"-A KUBEVIRT_POSTINBOUND --dport 23 -j SNAT --to-source 192.168.100.1",
		"-A KUBEVIRT_POSTINBOUND --to-source 192.168.100.1",
	}
	mockOutputRules = []string{
		"-A POSTROUTING -o k6t-eth0 -j KUBEVIRT_POSTINBOUND",
		"-A POSTROUTING -s 192.168.100.2/32 -j MASQUERADE",
	}
)

func TestParseForwardPorts(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockHandler := NewMockIptablesHandler(mockCtrl)
	mockHandler.EXPECT().
		IptablesListRules("test", "KUBEVIRT_POSTINBOUND").
		Return(mockForwardRules, nil).
		Times(1)

	ports, err := ParseForwardPorts(mockHandler, "test", "KUBEVIRT_POSTINBOUND")

	if err != nil {
		t.Errorf("Parse ports error: %s", err)
	}
	t.Logf("ports: %v", ports)
	if len(ports) != 2 || ports[0] != 22 || ports[1] != 23 {
		t.Fail()
	}
}

func TestParseVirtualMachineIP(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockHandler := NewMockIptablesHandler(mockCtrl)
	mockHandler.EXPECT().
		IptablesListRules("test", "POSTROUTING").
		Return(mockOutputRules, nil).
		Times(1)

	ipNet, err := ParseVirtualMachineIP(mockHandler, "test", "POSTROUTING")

	if err != nil {
		t.Errorf("Parse VM ip error: %s", err)
	}
	t.Logf("ip: %s", ipNet.String())
	if ipNet.String() != "192.168.100.2/32" {
		t.Fail()
	}
}
