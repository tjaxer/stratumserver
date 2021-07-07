/*
 * Copyright (c) 2020 The JaxNetwork developers
 * Use of this source code is governed by an ISC
 * license that can be found in the LICENSE file.
 */

package settings

import (
	//"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"runtime"
	//"strconv"
	//"strings"
	//"time"

	//"gitlab.com/jaxnet/core/miner/core/common"
	"github.com/tjaxer/stratumserver/e"
	//"github.com/tjaxer/stratumserver/logger"
	//"gitlab.com/jaxnet/jaxnetd/jaxutil"
	"gitlab.com/jaxnet/jaxnetd/network/rpcclient"
	//"gitlab.com/jaxnet/jaxnetd/types/chaincfg"
	"gitlab.com/jaxnet/jaxnetd/types/jaxjson"
	"gopkg.in/yaml.v3"
)

type Observer struct {
	outgoingConfigurations chan *Configuration

	config   *Configuration
	settings *Settings

	cachedShardsResponse *jaxjson.ShardListResult
}

func NewObserver() (observer *Observer, err error) {
	observer = &Observer{
		outgoingConfigurations: make(chan *Configuration),
	}

	err = observer.loadInitialConfig()
	return
}

//func (o *Observer) ObserveConfigurationChanges() {
//	mode := o.settings.Network.Shards.Discovering.Dynamic.Mode
//	if mode == "off" {
//		// report configuration once and exit.
//		o.outgoingConfigurations <- o.config
//		return
//	}
//
//	for {
//		if mode == "once" || mode == "periodically" {
//			shards, err := o.fetchShardsInfo()
//			if err != nil {
//				if err != e.ErrDuplicatedResponse {
//					logger.Log.Err(err).Msg("Can't fetch shards info from node. Would try again on the next round")
//				}
//
//				goto loopEnd
//			}
//
//			newConf, err := o.buildConfig(shards)
//			if newConf != nil {
//				o.config = newConf
//				o.outgoingConfigurations <- newConf
//			}
//		}
//
//	loopEnd:
//		if mode == "once" {
//			// No more conf. should be fetched.
//			return
//		}
//		time.Sleep(time.Second * 10) // todo: https://gitlab.com/jaxnet/core/miner/-/issues/5
//	}
//}

func (o *Observer) ConfigurationsFlow() (flow <-chan *Configuration) {
	flow = o.outgoingConfigurations
	return
}

func (o *Observer) loadInitialConfig() (err error) {
	path := flag.String("config", "conf.yaml", "path to configuration file")
	flag.Parse()

	data, err := ioutil.ReadFile(*path)
	if err != nil {
		return
	}

	o.settings = &Settings{}
	err = yaml.Unmarshal(data, o.settings)
	if err != nil {
		return
	}
	if o.settings.NumCPU > 0 {
		runtime.GOMAXPROCS(o.settings.NumCPU)
	}

	// Validation
	mode := o.settings.Network.Shards.Discovering.Dynamic.Mode
	if mode != "once" && mode != "periodically" && mode != "off" {
		err = fmt.Errorf("invalid option occured in `Network.Shards.Discovering.Dynamic.Mode`, " +
			"available options are: once, periodically, off")
		return
	}
	// ToDo: Restore config
	//o.config, err = o.buildConfig(map[uint32]jaxjson.ShardInfo{})

	// ToDo: Add config validation.
	//		 (https://gitlab.com/jaxnet/core/miner/-/issues/5)
	return
}

func (o *Observer) fetchShardsInfo() (shards map[uint32]jaxjson.ShardInfo, err error) {
	client, err := rpcclient.New(o.config.BeaconRPCConf(), nil)
	if err != nil {
		return
	}

	response, err := client.ListShards()
	if err != nil {
		return
	}

	defer func() {
		o.cachedShardsResponse = response
	}()

	// todo: deduplication (add comment)
	if o.cachedShardsResponse != nil {
		if len(response.Shards) != len(o.cachedShardsResponse.Shards) {
			// Received shards response contains new data and must be propagated.
			shards = response.Shards
			return
		}

		for cachedShardID := range o.cachedShardsResponse.Shards {
			_, isPresent := response.Shards[cachedShardID]
			if !isPresent {
				// Received shards response contains new data and must be propagated.
				shards = response.Shards
				return
			}
		}

		err = e.ErrDuplicatedResponse
	} else {
		shards = response.Shards
	}

	return
}

//func (o *Observer) buildConfig(receivedShards map[uint32]jaxjson.ShardInfo) (newConf *Configuration, err error) {
//	newConf = &Configuration{}
//	newConf.Debug = o.settings.Debug
//	newConf.Network.Name = o.settings.Network.Name
//	newConf.Network.Beacon.Discovering.RPC = o.settings.Network.Beacon.Discovering.RPC
//	newConf.Network.Shards = make(map[common.ShardID]ShardConfig)
//
//	// Applying fetched shards
//	for id := range receivedShards {
//		shardConf := ShardConfig{}
//		shardConf.ID = common.ShardID(id)
//		shardConf.RPC = o.settings.Network.Beacon.Discovering.RPC
//		newConf.Network.Shards[shardConf.ID] = shardConf
//	}
//
//	// Merging static settings.
//	// This kind of settings has greater priority, so must rewrite shard, in case if already present.
//	for _, conf := range o.settings.Network.Shards.Discovering.Static.Credentials {
//		newConf.Network.Shards[conf.ID] = conf
//	}
//
//	// Switch to enable merge mining with bitcoin
//	newConf.EnableStratum = o.settings.EnableStratum
//
//	// Stratum server config
//	newConf.Stratum.Banning = struct {
//		Enabled        bool
//		Time           int
//		InvalidPercent float64
//		CheckThreshold uint64
//		PurgeInterval  int
//	}{
//		Enabled:        o.settings.Stratum.Banning.Enabled,
//		Time:           o.settings.Stratum.Banning.Time,
//		InvalidPercent: o.settings.Stratum.Banning.InvalidPercent,
//		CheckThreshold: o.settings.Stratum.Banning.CheckThreshold,
//		PurgeInterval:  o.settings.Stratum.Banning.PurgeInterval,
//	}
//
//	newConf.Stratum.JobRebroadcastTimeout = o.settings.Stratum.JobRebroadcastTimeout
//	newConf.Stratum.TcpProxyProtocol = o.settings.Stratum.TcpProxyProtocol
//	newConf.Stratum.ConnectionTimeout = o.settings.Stratum.ConnectionTimeout
//
//	newConf.Stratum.Ports.Port = o.settings.Stratum.Ports.Port
//	newConf.Stratum.Ports.Diff = o.settings.Stratum.Ports.Diff
//	newConf.Stratum.Ports.TLS = o.settings.Stratum.Ports.TLS
//
//	newConf.Stratum.Ports.VarDiff = struct {
//		MinDiff         float64
//		MaxDiff         float64
//		TargetTime      int64
//		RetargetTime    int64
//		VariancePercent float64
//		X2Mode          bool
//	}{
//		MinDiff:         o.settings.Stratum.Ports.VarDiff.MinDiff,
//		MaxDiff:         o.settings.Stratum.Ports.VarDiff.MaxDiff,
//		TargetTime:      o.settings.Stratum.Ports.VarDiff.TargetTime,
//		RetargetTime:    o.settings.Stratum.Ports.VarDiff.RetargetTime,
//		VariancePercent: o.settings.Stratum.Ports.VarDiff.VariancePercent,
//		X2Mode:          o.settings.Stratum.Ports.VarDiff.X2Mode,
//	}
//
//	net := chaincfg.NetName(newConf.Network.Name).Params()
//	newConf.EnableBTCMining = o.settings.EnableBTCMining
//	newConf.BtcMiningAddress, err = jaxutil.DecodeAddress(o.settings.BtcMiningAddress, net)
//	if err != nil {
//		return
//	}
//	newConf.JaxMiningAddress, err = jaxutil.DecodeAddress(o.settings.JaxMiningAddress, net)
//	if err != nil {
//		return
//	}
//
//	if !o.settings.BurnJaxReward && !o.settings.BurnBtcReward {
//		err = errors.New("to emit the JaxNet Coins, need to burn BTC ")
//		return
//	}
//	if !o.settings.BurnJaxNetReward && !o.settings.BurnJaxReward && !o.settings.BurnBtcReward {
//		err = errors.New("to emit the Jax Coins, need to burn Jax Coins and BTC ")
//		return
//	}
//
//	newConf.BurnJaxNetReward = o.settings.BurnJaxNetReward
//	newConf.BurnJaxReward = o.settings.BurnJaxReward
//	newConf.BurnBtcReward = o.settings.BurnBtcReward
//
//	newConf.Network.Bitcoin.Params = o.settings.Network.Bitcoin.Params
//	newConf.Network.Bitcoin.Network = o.settings.Network.Bitcoin.Network
//	newConf.Network.Bitcoin.User = o.settings.Network.Bitcoin.User
//	newConf.Network.Bitcoin.Pass = o.settings.Network.Bitcoin.Pass
//	newConf.Network.Bitcoin.ExtraHeaders = o.settings.Network.Bitcoin.ExtraHeaders
//
//	o.filterConfigurationAndLogErrors(newConf)
//	return
//}

//func (o *Observer) filterConfigurationAndLogErrors(conf *Configuration) {
//	tokens := strings.Split(o.settings.Network.Shards.Discovering.Static.Forbidden, ",")
//	for _, token := range tokens {
//		if len(token) >= 6 &&
//			token[0] == '(' &&
//			token[len(token)-1] == ')' &&
//			strings.Contains(token, "..") {
//
//			ranges := strings.Split(token, "..")
//			if len(ranges) == 2 {
//				from, err := strconv.Atoi(ranges[0])
//				if err != nil {
//					// todo: log error
//					continue
//				}
//
//				to, err := strconv.Atoi(ranges[1])
//				if err != nil {
//					// todo: log error
//					continue
//				}
//
//				for forbiddenID := common.ShardID(from); forbiddenID <= common.ShardID(to); forbiddenID++ {
//					delete(conf.Network.Shards, forbiddenID)
//				}
//			}
//
//		} else {
//			id, err := strconv.Atoi(token)
//			if err != nil {
//				continue
//			}
//
//			delete(conf.Network.Shards, common.ShardID(id))
//		}
//	}
//}
