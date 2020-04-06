package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/weblazy/core/iptables"
	utiliptables "github.com/weblazy/core/iptables"
	"k8s.io/klog"
	"k8s.io/utils/exec"
)

const (
	// the hostport chain
	kubeHostportsChain utiliptables.Chain = "KUBE-HOSTPORTS"
	// prefix for hostport chains
	kubeHostportChainPrefix string = "KUBE-HP-"
)

// @desc
// @auth liuguoqiang 2020-04-06
// @param
// @return
func main() {
	fmt.Println("main")
	execer := exec.New()
	iptInterface := iptables.New(execer, utiliptables.ProtocolIpv4)
	EnsureRule(iptInterface)
	Delete(iptInterface)
	getExistingHostportIPTablesRules(iptInterface)
	// 	chain, rule, err := getExistingHostportIPTablesRules(iptInterface)
	// 	fmt.Printf("chain:%#v\n", chain)
	// 	fmt.Printf("rule:%#v\n", rule)
	// 	fmt.Printf("err:%#v\n", err)
}

func getExistingHostportIPTablesRules(iptables utiliptables.Interface) (map[utiliptables.Chain]string, []string, error) {
	iptablesData := bytes.NewBuffer(nil)
	err := iptables.SaveInto(utiliptables.TableNAT, iptablesData)
	if err != nil { // if we failed to get any rules
		return nil, nil, fmt.Errorf("failed to execute iptables-save: %v", err)
	}
	existingNATChains := utiliptables.GetChainLines(utiliptables.TableNAT, iptablesData.Bytes())
	existingHostportChains := make(map[utiliptables.Chain]string)
	existingHostportRules := []string{}

	for chain := range existingNATChains {
		if strings.HasPrefix(string(chain), string(kubeHostportsChain)) || strings.HasPrefix(string(chain), kubeHostportChainPrefix) {
			existingHostportChains[chain] = string(existingNATChains[chain])
			fmt.Printf("%s:%s\n", chain, string(existingNATChains[chain]))
		}
	}

	for _, line := range strings.Split(iptablesData.String(), "\n") {
		if strings.HasPrefix(line, fmt.Sprintf("-A %s", "PREROUTING")) ||
			strings.HasPrefix(line, fmt.Sprintf("-A %s", "OUTPUT")) {
			existingHostportRules = append(existingHostportRules, line)
			fmt.Printf("%s\n", line)
		}
	}
	return existingHostportChains, existingHostportRules, nil
}

// @desc
// @auth liuguoqiang 2020-04-06
// @param
// @return
func EnsureRule(iptables utiliptables.Interface) {
	if _, err := iptables.EnsureRule(utiliptables.Append, utiliptables.TableNAT, "PREROUTING", "-p", "tcp", "-i", "eth0", "--dport", "2077", "-j", "DNAT", "--to", "10.0.0.6:2077"); err != nil {
		klog.Errorf("%#v", err)
		return
	}
}

// @desc
// @auth liuguoqiang 2020-04-06
// @param
// @return
func Delete(iptables utiliptables.Interface) {
	if err := iptables.DeleteRule(utiliptables.TableNAT, utiliptables.ChainPrerouting, "-p", "tcp", "-i", "eth0", "--dport", "2077", "-j", "DNAT"); err != nil {
		if !utiliptables.IsNotFoundError(err) {
			klog.Errorf("Error removing userspace rule: %v", err)
		}
	}
}
