/*
 * Copyright (c) 2020 The JaxNetwork developers
 * Use of this source code is governed by an ISC
 * license that can be found in the LICENSE file.
 */

package utils

import (
	"sync"
	"time"
)

const (
	stoppingTTL = time.Millisecond * 50
)

type StoppableMixin struct {
	MustBeStopped bool

	IsStopped func() bool
}

func (sm *StoppableMixin) StopUsing(wg *sync.WaitGroup) {
	sm.MustBeStopped = true

	for {
		time.Sleep(stoppingTTL)

		if sm.IsStopped() == false {
			continue

		} else {
			wg.Done()
			return
		}
	}
}
