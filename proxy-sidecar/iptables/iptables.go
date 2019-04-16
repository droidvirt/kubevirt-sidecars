package iptables

import (
	"strings"

	"github.com/coreos/go-iptables/iptables"
	"github.com/spf13/pflag"
)

var (
	// Handler :
	Handler              IptablesHandler
	rulesTable           = "nat"
	kubevirtForwardChain = "KUBEVIRT_POSTINBOUND"
)

// IptablesHandler create interface just for mock
type IptablesHandler interface {
	iptablesNewChain(table string, chain string) error
	iptablesClearChain(table string, chain string) error
	iptablesAppendRule(table string, chain string, rulespec ...string) error
	iptablesListRules(table string, chain string) ([]string, error)
	// ParseForwardPorts() ([]int, error)
}

type IptablesUtilsHandler struct{}

// iptables -t table -N chain
func (h *IptablesUtilsHandler) iptablesNewChain(table string, chain string) error {
	iptablesObject, err := iptables.New()
	if err != nil {
		return err
	}
	return iptablesObject.NewChain(table, chain)
}

// iptables -t table -F chain
func (h *IptablesUtilsHandler) iptablesClearChain(table string, chain string) error {
	iptablesObject, err := iptables.New()
	if err != nil {
		return err
	}
	return iptablesObject.ClearChain(table, chain)
}

// iptables -t table -A chain ...
func (h *IptablesUtilsHandler) iptablesAppendRule(table string, chain string, rulespec ...string) error {
	iptablesObject, err := iptables.New()
	if err != nil {
		return err
	}
	return iptablesObject.Append(table, chain, rulespec...)
}

// iptables -t table -S chain
func (h *IptablesUtilsHandler) iptablesListRules(table string, chain string) ([]string, error) {
	iptablesObject, err := iptables.New()
	if err != nil {
		return nil, err
	}
	rules, err := iptablesObject.List(table, chain)
	if err != nil {
		return nil, err
	}
	return rules[1:], nil
}

// ParseForwardPorts :
// parse which ports be forwarded from libvirt VM to pod nic
func ParseForwardPorts(h IptablesHandler) ([]int, error) {
	rules, err := h.iptablesListRules(rulesTable, kubevirtForwardChain)
	if err != nil {
		return nil, err
	}

	ports := make([]int, 0)
	for _, rule := range rules {
		flags := pflag.NewFlagSet("iptables-flag", pflag.ContinueOnError)
		flags.ParseErrorsWhitelist.UnknownFlags = true
		forwardPort := flags.Int("dport", 0, "")
		err := flags.Parse(strings.Split(rule, " "))
		if err == nil {
			ports = append(ports, *forwardPort)
		} else {
			return nil, err
		}
	}

	return ports, nil
}
