/*
 * Copyright (c) 2020 The JaxNetwork developers
 * Use of this source code is governed by an ISC
 * license that can be found in the LICENSE file.
 */

package settings

import (
	"fmt"
)

type NetworkConfig struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

func (nc *NetworkConfig) Interface() string {
	return fmt.Sprint(nc.Host, ":", nc.Port)
}

func (nc *NetworkConfigBitcoin) Interface() string {
	// For non-TLS connections port is required.
	// For TLS connections port is hardcoded to 443.
	if nc.DisableTLS {
		return fmt.Sprint(nc.Host, ":", nc.Port)
	} else {
		return nc.Host
	}
}

type RPCConfig struct {
	User    string        `yaml:"user"`
	Pass    string        `yaml:"pass"`
	Network NetworkConfig `yaml:"network"`
}



type NetworkConfigBitcoin struct {
	Host       string `yaml:"host"`
	Port       uint16 `yaml:"port"`
	DisableTLS bool   `yaml:"disableTLS"`
}
