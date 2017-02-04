// +build !linux

// only tested on OSX
// decided to go with exec.Command after I couldn't figure
// out how to extract the arp cache out of the kernel with
// golang's syscall or Sysctl()
//
// ... Help appreciated :)

package arp

import (
	"os/exec"
	"strings"
)

func Table() ArpTable {
	data, err := exec.Command("arp", "-an").Output()
	if err != nil {
		return nil
	}

	var table = make(ArpTable)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		//get unbracketed ip address
		ip := strings.Trim(fields[1], "()")
		//get mac address
		mac := fields[3]
		if mac == "(incomplete)" {
			continue
		}
		//enforce 2 hex chars
		octs := strings.SplitN(mac, ":", 6)
		for i, oct := range octs {
			if len(oct) == 1 {
				octs[i] = "0" + oct
			}
		}
		mac = strings.Join(octs, ":")
		//store entry
		table[ip] = mac
	}

	return table
}
