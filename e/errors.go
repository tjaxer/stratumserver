/*
 * Copyright (c) 2020 The JaxNetwork developers
 * Use of this source code is governed by an ISC
 * license that can be found in the LICENSE file.
 */

package e

import (
	"errors"
	"log"
)

var (
	ErrInvalidShardID         = errors.New("invalid shard ID")
	ErrInvalidConfiguration   = errors.New("invalid configuration")
	ErrInvalidBlockHeaderBits = errors.New("invalid block header bits")
	ErrDuplicatedBlock        = errors.New("duplicated block")
	ErrNoBCHeader             = errors.New("no BC header")
	ErrDuplicatedResponse     = errors.New("duplicated response")
)

func AbortOn(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
