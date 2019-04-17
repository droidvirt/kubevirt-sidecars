package iptables

//go:generate mockgen -destination=./iptables_mock.go -package=iptables -write_package_comment=false kubevirt.io/kubevirt/cmd/droidvirt-sidecar/proxy-sidecar/iptables IptablesHandler

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	"github.com/spf13/pflag"
)

var (
	// Handler :
	Handler              IptablesHandler
)

// IptablesHandler : just for mock
type IptablesHandler interface {
	IptablesNewChain(table string, chain string) error
	IptablesClearChain(table string, chain string) error
	IptablesAppendRule(table string, chain string, rulespec ...string) error
	IptablesInsertRule(table string, chain string, rulespec ...string) error
	IptablesListRules(table string, chain string) ([]string, error)
}

type IptablesUtilsHandler struct{}

// iptables -t table -N chain
func (h *IptablesUtilsHandler) IptablesNewChain(table string, chain string) error {
	iptablesObject, err := iptables.New()
	if err != nil {
		return err
	}
	return iptablesObject.NewChain(table, chain)
}

// iptables -t table -F chain
func (h *IptablesUtilsHandler) IptablesClearChain(table string, chain string) error {
	iptablesObject, err := iptables.New()
	if err != nil {
		return err
	}
	return iptablesObject.ClearChain(table, chain)
}

// iptables -t table -A chain ...
func (h *IptablesUtilsHandler) IptablesAppendRule(table string, chain string, rulespec ...string) error {
	iptablesObject, err := iptables.New()
	if err != nil {
		return err
	}
	return iptablesObject.Append(table, chain, rulespec...)
}

// iptables -t table -I chain ...
func (h *IptablesUtilsHandler) IptablesInsertRule(table string, chain string, rulespec ...string) error {
	iptablesObject, err := iptables.New()
	if err != nil {
		return err
	}
	return iptablesObject.Insert(table, chain, 0, rulespec...)
}

// iptables -t table -S chain
func (h *IptablesUtilsHandler) IptablesListRules(table string, chain string) ([]string, error) {
	iptablesObject, err := iptables.New()
	if err != nil {
		return nil, err
	}
	rules, err := iptablesObject.List(table, chain)
	if err != nil {
		return nil, err
	}
	// ignore rule of chain creation
	return rules[1:], nil
}

func InitHandler() {
	if Handler == nil {
		Handler = &IptablesUtilsHandler{}
	}
}

// ParseForwardPorts :
// parse rule params of "--dport"
func ParseForwardPorts(h IptablesHandler, nat string, chain string) ([]int, error) {
	rules, err := h.IptablesListRules(nat, chain)
	if err != nil {
		return nil, err
	}

	reason := ""
	ports := make([]int, 0)
	for _, rule := range rules {
		flags := pflag.NewFlagSet("iptables-flag", pflag.ContinueOnError)
		flags.ParseErrorsWhitelist.UnknownFlags = true
		forwardPort := flags.Int("dport", 0, "")
		err := flags.Parse(strings.Split(rule, " "))
		if err != nil {
			reason = fmt.Sprintf("%s; %s", reason, err.Error())
		} else if *forwardPort != 0 {
			ports = append(ports, *forwardPort)
		}
	}

	return ports, nil
}

// ParseVirtualMachineIP :
// parse rule params of "--source" or "-s"
func ParseVirtualMachineIP(h IptablesHandler, nat string, chain string) (*net.IPNet, error) {
	rules, err := h.IptablesListRules(nat, chain)
	if err != nil {
		return nil, err
	}

	reason := ""
	for _, rule := range rules {
		flags := pflag.NewFlagSet("iptables-flag", pflag.ContinueOnError)
		flags.ParseErrorsWhitelist.UnknownFlags = true
		sourceCIDR := flags.StringP("source", "s", "", "")
		err := flags.Parse(strings.Split(rule, " "))
		if err != nil {
			reason = fmt.Sprintf("%s; %s", reason, err.Error())
		} else if *sourceCIDR != "" {
			_, ipNet, err := net.ParseCIDR(*sourceCIDR)
			if err == nil {
				return ipNet, nil
			} else {
				reason = fmt.Sprintf("%s; %s", reason, err.Error())
			}
		}
	}
	return nil, errors.New(reason)
}