package server

import (
	"encoding/binary"
	"math"
)

// SubscriptionCounter describes number of currently connected stratum clients.
// Supports 18446744073709551615 max connections.
type SubscriptionCounter struct {
	Count   uint64
	Padding []byte
}

func NewSubscriptionCounter() *SubscriptionCounter {
	return &SubscriptionCounter{
		Count:   0,
		Padding: nil,
	}
}

func (sc *SubscriptionCounter) Next() []byte {
	sc.Count++
	if sc.Count == math.MaxUint64 {
		sc.Count = 0
	}

	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, sc.Count)
	return append(sc.Padding, b...)
}
