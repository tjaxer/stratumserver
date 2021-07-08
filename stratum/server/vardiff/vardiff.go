package vardiff

import (
	"time"
)

type VarDiff struct {
	BufferSize    int64
	MaxTargetTime float64
	MinTargetTime float64

	TimeBuffer    *RingBuffer
	LastRtc       int64
	LastTimestamp int64

	MinDiff         float64
	MaxDiff         float64
	TargetTime      int64
	RetargetTime    int64
	VariancePercent float64
	X2Mode          bool
}

func NewVarDiff(MinDiff float64,
	MaxDiff float64,
	TargetTime int64,
	RetargetTime int64,
	VariancePercent float64,
	X2Mode bool) *VarDiff {
	timestamp := time.Now().Unix()
	bufferSize := RetargetTime / TargetTime * 4
	return &VarDiff{
		BufferSize:    bufferSize,
		MaxTargetTime: float64(TargetTime) * (1 + VariancePercent),
		MinTargetTime: float64(TargetTime) * (1 - VariancePercent),
		TimeBuffer:    NewRingBuffer(bufferSize),
		LastRtc:       timestamp - RetargetTime/2,
		LastTimestamp: timestamp,

		MinDiff:         MinDiff,
		MaxDiff:         MaxDiff,
		TargetTime:      TargetTime,
		RetargetTime:    RetargetTime,
		VariancePercent: VariancePercent,
		X2Mode:          X2Mode,
	}
}

func (vd *VarDiff) CalcNextDiff(currentDiff float64) (newDiff float64) {
	timestamp := time.Now().Unix()

	if vd.LastRtc == 0 {
		vd.LastRtc = timestamp - vd.RetargetTime/2
		vd.LastTimestamp = timestamp
		return
	}

	sinceLast := timestamp - vd.LastTimestamp

	vd.TimeBuffer.Append(sinceLast)
	vd.LastTimestamp = timestamp

	if (timestamp-vd.LastRtc) < vd.RetargetTime && vd.TimeBuffer.Size() > 0 {
		return
	}

	vd.LastRtc = timestamp
	avg := vd.TimeBuffer.Avg()
	ddiff := float64(time.Duration(vd.TargetTime)*time.Second) / avg

	// currentDiff, _ := client.Difficulty.Float64()

	if avg > vd.MaxTargetTime && currentDiff > vd.MinDiff {
		if vd.X2Mode {
			ddiff = 0.5
		}

		if ddiff*currentDiff < vd.MinDiff {
			ddiff = vd.MinDiff / currentDiff
		}
	} else if avg < vd.MinTargetTime {
		if vd.X2Mode {
			ddiff = 2
		}

		diffMax := vd.MaxDiff

		if ddiff*currentDiff > diffMax {
			ddiff = diffMax / currentDiff
		}
	} else {
		return currentDiff
	}

	newDiff = currentDiff * ddiff

	if newDiff <= 0 {
		newDiff = currentDiff
	}

	vd.TimeBuffer.Clear()
	return
}
