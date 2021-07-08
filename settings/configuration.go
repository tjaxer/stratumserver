/*
 * Copyright (c) 2020 The JaxNetwork developers
 * Use of this source code is governed by an ISC
 * license that can be found in the LICENSE file.
 */

package settings

import (
	btcdrpcclient "github.com/btcsuite/btcd/rpcclient"
	//"gitlab.com/jaxnet/core/miner/core/common"
	//"github.com/tjaxer/stratumserver/e"
	"gitlab.com/jaxnet/jaxnetd/jaxutil"
	//"gitlab.com/jaxnet/jaxnetd/network/rpcclient"
)

type Configuration struct {
	Debug  bool
	NumCPU int

	Network struct {
		Name string
		//Beacon struct {
		//	Discovering struct {
		//		RPC RPCConfig
		//	}
		//}
		//Shards map[common.ShardID]ShardConfig

		Bitcoin struct {
			Params       string
			Network      NetworkConfigBitcoin
			User         string
			Pass         string
			ExtraHeaders map[string]string
		}
	}

	EnableStratum bool

	Stratum struct {
		Banning struct {
			Enabled        bool
			Time           int
			InvalidPercent float64
			CheckThreshold uint64
			PurgeInterval  int
		}

		JobRebroadcastTimeout int
		TcpProxyProtocol      bool
		ConnectionTimeout     int

		Ports struct {
			Port    string
			Diff    float64
			VarDiff struct {
				MinDiff         float64
				MaxDiff         float64
				TargetTime      int64
				RetargetTime    int64
				VariancePercent float64
				X2Mode          bool
			}
			TLS bool
		}
	}

	EnableBTCMining bool

	//BurnBtcReward    bool
	//BurnJaxReward    bool
	//BurnJaxNetReward bool
	BtcMiningAddress jaxutil.Address
	//JaxMiningAddress jaxutil.Address
}

// TotalNetworkShardsCount returns the total amount of shards in the network
// (according to the config).
// The total amount of shards that are listed in config file could differ
// from the total amount of shards that are present in the network,
// e.g. miner could mine shards 100, and 200 and the total amount of shards mined == 2,
// but the total amount of shards in the network, according to this config is 200.

//func (s *Configuration) TotalNetworkShardsCount() uint32 {
//	return uint32(len(s.Network.Shards))
//}

// BeaconRPCConf returns RPC connection parameters of the beacon chain
// (in a format that is appropriate for the RPC-client).
//func (s *Configuration) BeaconRPCConf() (conf *rpcclient.ConnConfig) {
//	beacon := s.Network.Beacon.Discovering.RPC
//	conf = s.defaultRPCConfig()
//	conf.Host = beacon.Network.Interface()
//	conf.User = beacon.User
//	conf.Pass = beacon.Pass
//	return
//}

// ShardRPCConf returns RPC connection parameters of the specified shard chain.
// (in a format that is appropriate for the RPC-client).
//func (s *Configuration) ShardRPCConf(shardID common.ShardID) (conf *rpcclient.ConnConfig, err error) {
//	for _, shard := range s.Network.Shards {
//		if shard.ID == shardID {
//			rpc := shard.RPC
//			conf = s.defaultRPCConfig()
//			conf.Host = rpc.Network.Interface()
//			conf.User = rpc.User
//			conf.Pass = rpc.Pass
//			return
//		}
//	}
//
//	err = e.ErrInvalidShardID
//	return
//}

// defaultRPCConfig returns RPC connection parameters
// that are common for beacon chain and all shard chains.
//func (s *Configuration) defaultRPCConfig() *rpcclient.ConnConfig {
//	return &rpcclient.ConnConfig{
//		Params:       s.Network.Name,
//		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
//		DisableTLS:   true, // Bitcoin core does not provide TLS by default
//	}
//}

// BitcoinRPCConf returns RPC connection parameters of the bitcoin node
// (in a format that is appropriate for the RPC-client).
func (s *Configuration) BitcoinRPCConf() (conf *btcdrpcclient.ConnConfig) {
	bitcoin := s.Network.Bitcoin
	conf = s.defaultBitcoinRPCConfig()
	conf.Params = bitcoin.Params
	conf.Host = bitcoin.Network.Interface()
	conf.User = bitcoin.User
	conf.Pass = bitcoin.Pass
	conf.DisableTLS = bitcoin.Network.DisableTLS
	conf.ExtraHeaders = bitcoin.ExtraHeaders

	return
}

// defaultBitcoinRPCConfig returns RPC connection parameters
// that are common for bitcoin chain.
func (s *Configuration) defaultBitcoinRPCConfig() *btcdrpcclient.ConnConfig {
	return &btcdrpcclient.ConnConfig{
		Params:       s.Network.Bitcoin.Params,
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
	}
}

type Settings struct {
	Debug           bool `yaml:"debug"`
	NumCPU          int  `yaml:"num_cpu"`
	EnableBTCMining bool `yaml:"enable_btc_mining"`
	//BurnBtcReward    bool   `yaml:"burn_btc"`
	//BurnJaxReward    bool   `yaml:"burn_jax"`
	//BurnJaxNetReward bool   `yaml:"burn_jaxnet"`
	BtcMiningAddress string `yaml:"btc_mining_address"`
	//JaxMiningAddress string `yaml:"jax_mining_address"`

	Network struct {
		Name string `yaml:"name"`

		//Beacon struct {
		//	Discovering struct {
		//		RPC RPCConfig `yaml:"rpc"`
		//	} `yaml:"discovering"`
		//} `yaml:"beacon"`
		//
		//Shards struct {
		//	Discovering struct {
		//		Dynamic struct {
		//			Mode      string `yaml:"mode"`
		//			RoundTime string `yaml:"round-time"`
		//		} `yaml:"dynamic"`
		//
		//		Static struct {
		//			Credentials []ShardConfig `yaml:"credentials"`
		//			Forbidden   string        `yaml:"forbidden"`
		//		} `yaml:"static"`
		//	} `yaml:"discovering"`
		//} `yaml:"shards"`

		Bitcoin struct {
			Params       string               `yaml:"params"`
			Network      NetworkConfigBitcoin `yaml:"network"`
			User         string               `yaml:"user"`
			Pass         string               `yaml:"pass"`
			ExtraHeaders map[string]string    `yaml:"extra_headers"`
		} `yaml:"bitcoin"`
	} `yaml:"network"`

	EnableStratum bool `yaml:"enable_stratum"`

	Stratum struct {
		Banning struct {
			Enabled        bool    `yaml:"enabled"`
			Time           int     `yaml:"time"`
			InvalidPercent float64 `yaml:"invalid_percent"`
			CheckThreshold uint64  `yaml:"check_threshold"`
			PurgeInterval  int     `yaml:"purge_interval"`
		} `yaml:"banning"`

		JobRebroadcastTimeout int  `yaml:"job_rebroadcast_timeout"`
		TcpProxyProtocol      bool `yaml:"tcp_proxy_protocol"`
		ConnectionTimeout     int  `yaml:"connection_timeout"`

		Ports struct {
			Port    string  `yaml:"port"`
			Diff    float64 `yaml:"diff"`
			VarDiff struct {
				MinDiff         float64 `yaml:"min_diff"`
				MaxDiff         float64 `yaml:"max_diff"`
				TargetTime      int64   `yaml:"target_time"`
				RetargetTime    int64   `yaml:"retarget_time"`
				VariancePercent float64 `yaml:"variance_percent"`
				X2Mode          bool    `yaml:"x2_mode"`
			} `yaml:"var_diff"`
			TLS bool `yaml:"tls"`
		} `yaml:"ports"`
	} `yaml:"stratum"`
}
