package main

import (
	"errors"
	"fmt"
	"kubevirt.io/kubevirt/cmd/droidvirt-sidecar/proxy-sidecar/iptables"
)

const (
	rulesTable           = "nat"
	preroutingChain      = "PREROUTING"
	postroutingChain     = "POSTROUTING"
	outputChain          = "OUTPUT"
	kubevirtForwardChain = "KUBEVIRT_PREINBOUND"
	socksChain           = "SOCKS"
	podNIC               = "eth0"
	kubevirtBridgeNIC    = "k6t-eth0"
)

// modify iptable rules created by virt-launcher
// redirToAddress, redirToPort: redir packet from libvirt VM to port
func hackIptables(redirToAddress string, redirToPort int) error {
	iptables.InitHandler()

	vmIPNet, err := iptables.ParseVirtualMachineIP(iptables.Handler, rulesTable, postroutingChain)
	if err != nil {
		return err
	} else if vmIPNet == nil {
		return errors.New("can not parse VM ip from iptable rules")
	}

	//forwardPorts, err := iptables.ParseForwardPorts(iptables.Handler, rulesTable, kubevirtForwardChain)
	//if err != nil {
	//	return err
	//}

	// SOCKS chain
	err = iptables.Handler.IptablesNewChain(rulesTable, socksChain)
	if err != nil {
		return err
	}
	err = iptables.Handler.IptablesAppendRule(rulesTable, socksChain, "-d", "127.0.0.0/8", "-j", "RETURN")
	if err != nil {
		return err
	}
	err = iptables.Handler.IptablesAppendRule(rulesTable, socksChain, "-p", "tcp",
		"-j", "DNAT", "--to-destination", fmt.Sprintf("%s:%d", redirToAddress, redirToPort))
	if err != nil {
		return err
	}

	// PREROUTING chain
	err = iptables.Handler.IptablesClearChain(rulesTable, preroutingChain)
	if err != nil {
		return err
	}
	err = iptables.Handler.IptablesAppendRule(rulesTable, preroutingChain, "-i", kubevirtBridgeNIC, "-j", socksChain)
	if err != nil {
		return err
	}
	err = iptables.Handler.IptablesAppendRule(rulesTable, preroutingChain, "-i", podNIC, "-j", kubevirtForwardChain)
	if err != nil {
		return err
	}

	// OUTPUT chain
	err = iptables.Handler.IptablesClearChain(rulesTable, outputChain)
	if err != nil {
		return err
	}

	// POSTROUTING chain
	err = iptables.Handler.IptablesClearChain(rulesTable, postroutingChain)
	if err != nil {
		return err
	}
	err = iptables.Handler.IptablesAppendRule(rulesTable, postroutingChain, "-s", vmIPNet.String(),
		"-o", podNIC, "-j", "MASQUERADE")
	if err != nil {
		return err
	}

	return nil
}
