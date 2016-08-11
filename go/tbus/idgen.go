package tbus

import (
	"github.com/willf/bitset"
)

const (
	// BucketBits is the total bits per bucket
	BucketBits = 256
	// BucketShift is the bits to shift converting bucket index to bit index
	BucketShift = uint(8)

	allSetBits = 0xffffffffffffffff
)

// BitsBucket creates a bucket
func BitsBucket() *bitset.BitSet {
	return bitset.From([]uint64{allSetBits, allSetBits, allSetBits, allSetBits})
}

// MinIDGen always generate the ID with minimum value
type MinIDGen struct {
	buckets []*bitset.BitSet
}

// Alloc generates an ID
func (g *MinIDGen) Alloc() uint32 {
	emptyBkt := 0
	for ; emptyBkt < len(g.buckets); emptyBkt++ {
		bkt := g.buckets[emptyBkt]
		if bkt == nil || !bkt.None() {
			break
		}
	}

	var bkt *bitset.BitSet
	if emptyBkt < len(g.buckets) {
		bkt = g.buckets[emptyBkt]
	}
	if bkt == nil {
		bkt = g.newBucket(emptyBkt)
	}
	index, found := bkt.NextSet(0)
	if !found {
		panic("BUG: bucket should not be full")
	}
	bkt.SetTo(index, false)
	return (uint32(emptyBkt) << BucketShift) + uint32(index)
}

// Release returns an ID
func (g *MinIDGen) Release(id uint32) {
	bktIndex := int(id >> BucketShift)
	if bktIndex >= len(g.buckets) {
		return
	}
	bkt := g.buckets[bktIndex]
	if bkt == nil {
		return
	}
	bktOff := id & (BucketBits - 1)
	bkt.SetTo(uint(bktOff), true)
	if bkt.All() {
		g.buckets[bktIndex] = nil
	}
}

func (g *MinIDGen) newBucket(index int) *bitset.BitSet {
	for index >= len(g.buckets) {
		g.buckets = append(g.buckets, nil)
	}
	bkt := BitsBucket()
	g.buckets[index] = bkt
	return bkt
}
