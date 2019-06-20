package wt

import (
	"encoding/binary"
	"io"
	"reflect"
	"unsafe"

	"github.com/bpot/bv"
	"github.com/bpot/rrr"
)

// WT is a static wavelet tree
type WT struct {
	bits        *rrr.RRR
	alphabet    []byte
	nodeOffsets []uint64
	nodeRank1s  []uint64
}

// New builds a wavelet tree representation of the sequence s
func New(s []byte) (*WT, error) {
	// Find minimum and maximum values.
	usedSymbols := make([]bool, 256)

	var min, max byte
	min = 255
	max = 0
	for _, ch := range s {
		usedSymbols[ch] = true
		if ch < min {
			min = ch
		}
		if ch > max {
			max = ch
		}
	}

	alphabet := []byte{}
	for ch, used := range usedSymbols {
		if used {
			alphabet = append(alphabet, byte(ch))
		}
	}

	bits, nodeOffsets := buildTree(s, alphabet)
	bitsRRR, err := rrr.NewFromBitVector(bits)
	if err != nil {
		return nil, err
	}
	wt := &WT{
		alphabet:    alphabet,
		bits:        bitsRRR,
		nodeOffsets: nodeOffsets,
	}
	wt.populateNodeRanks()
	return wt, nil
}

// Access returns the ith byte in the sequence
func (w *WT) Access(i uint64) byte {
	alphabet := w.alphabet
	node := 1
	rank := i
	for {

		offset := w.nodeOffsets[node-1]
		midIdx := (len(alphabet) + 1) / 2
		if w.bits.Access(offset + rank) {
			// RIGHT
			rankBase := w.nodeRank1s[node-1]
			rank = w.bits.Rank1(offset+rank) - rankBase
			node = node*2 + 1
			if node-1 >= len(w.nodeOffsets) {
				return alphabet[len(alphabet)-1]
			}
			alphabet = alphabet[midIdx:]
		} else {
			// LEFT
			rankBase := offset - w.nodeRank1s[node-1]
			rank = w.bits.Rank0(offset+rank) - rankBase
			node = node * 2
			if node-1 >= len(w.nodeOffsets) {
				return alphabet[0]
			}
			alphabet = alphabet[:midIdx]
		}
	}
}

// Rank returns the rank of the symbole c at position i
func (w *WT) Rank(c byte, i uint64) uint64 {
	alphabet := w.alphabet
	found := false
	for _, ch := range alphabet {
		if ch == c {
			found = true
		}
	}
	if !found {
		return 0
	}
	node := 1
	rank := i
	for {
		offset := w.nodeOffsets[node-1]
		midIdx := (len(alphabet) + 1) / 2

		if len(alphabet) == 1 {
			return rank
		}

		if c < alphabet[midIdx] {
			rankBase := offset - w.nodeRank1s[node-1]
			rank = w.bits.Rank0(offset+rank) - rankBase
			node = node * 2
			if len(alphabet[:midIdx]) == 1 {
				return rank
			}
			alphabet = alphabet[:midIdx]
		} else {
			rankBase := w.nodeRank1s[node-1]
			rank = w.bits.Rank1(offset+rank) - rankBase
			node = node*2 + 1
			if len(alphabet[midIdx:]) == 1 {
				return rank
			}
			alphabet = alphabet[midIdx:]
		}
	}
}

// InverseSelect returns the symbol and rank of the symbol at position i
func (w *WT) InverseSelect(i uint64) (c byte, rank uint64) {
	alphabet := w.alphabet
	node := 1
	rank = i

	for {
		offset := w.nodeOffsets[node-1]
		midIdx := (len(alphabet) + 1) / 2
		if w.bits.Access(offset + rank) {
			rankFull := w.bits.Rank1(offset + rank)
			rankBase := w.nodeRank1s[node-1]
			rank = rankFull - rankBase
			node = node*2 + 1
			alphabet = alphabet[midIdx:]
			if len(alphabet) == 1 {
				return alphabet[0], rank
			}
		} else {
			rankFull := w.bits.Rank0(offset + rank)
			rankBase := offset - w.nodeRank1s[node-1]
			rank = rankFull - rankBase
			node = node * 2
			alphabet = alphabet[:midIdx]
			if len(alphabet) == 1 {
				return alphabet[0], rank
			}
		}
	}
}

// BitmapSize is the size of the compressed bitmap
func (w *WT) BitmapSize() int {
	return w.bits.Size()
}

// WriteTo writes a serialized version of the wavelet tree to writer
func (w *WT) WriteTo(writer io.Writer) (err error) {
	// Write alphabet size as uint64
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(len(w.alphabet)))
	_, err = writer.Write(buf)
	if err != nil {
		return err
	}

	// Write alphabet
	_, err = writer.Write(w.alphabet)
	if err != nil {
		return err
	}

	// Write nodeOffsets size
	binary.LittleEndian.PutUint64(buf, uint64(len(w.nodeOffsets)))
	_, err = writer.Write(buf)
	if err != nil {
		return err
	}

	// Write nodeOffsets
	_, err = writer.Write(uint64SliceAsByteSlice(w.nodeOffsets))
	if err != nil {
		return err
	}

	// Write rank1s size
	binary.LittleEndian.PutUint64(buf, uint64(len(w.nodeRank1s)))
	_, err = writer.Write(buf)
	if err != nil {
		return err
	}

	// Write rank1s
	_, err = writer.Write(uint64SliceAsByteSlice(w.nodeRank1s))
	if err != nil {
		return err
	}

	err = w.bits.WriteTo(writer)
	if err != nil {
		return err
	}

	return nil
}

func (wt *WT) populateNodeRanks() {
	wt.nodeRank1s = make([]uint64, len(wt.nodeOffsets))
	for i, off := range wt.nodeOffsets {
		wt.nodeRank1s[i] = wt.bits.Rank1(off)
	}
}

// NewFromSerialized returns a new wavelet tree for the serialized representation
func NewFromSerialized(buf []byte) (wt *WT, wtBytes int, err error) {
	wt = &WT{}

	// Set alphabet
	alphabetSize := int(binary.LittleEndian.Uint64(buf))
	buf = buf[8:]
	wtBytes += 8

	wt.alphabet = buf[:alphabetSize]
	buf = buf[alphabetSize:]
	wtBytes += alphabetSize

	// Set nodeOffsets
	rankSize := int(binary.LittleEndian.Uint64(buf))
	buf = buf[8:]
	wtBytes += 8

	wt.nodeOffsets = byteSliceAsUint64Slice(buf[:8*rankSize])
	buf = buf[8*rankSize:]
	wtBytes += 8 * rankSize

	nodeRank1sSize := int(binary.LittleEndian.Uint64(buf))
	buf = buf[8:]
	wtBytes += 8

	wt.nodeRank1s = byteSliceAsUint64Slice(buf[:8*nodeRank1sSize])
	buf = buf[8*nodeRank1sSize:]
	wtBytes += 8 * nodeRank1sSize

	// Set bits
	var rrrBytes int
	wt.bits, rrrBytes, err = rrr.NewFromSerialized(buf)
	if err != nil {
		return nil, 0, err
	}
	wtBytes += rrrBytes

	return wt, wtBytes, err
}

type n struct {
	s        []byte
	alphabet []byte
}

func buildTree(s []byte, alphabet []byte) (*bv.BV, []uint64) {
	// A wavelet tree will use at most n*log(sigma)
	bits := bv.New((log2(uint64(len(alphabet))) + 1) * len(s))
	nodeOffsets := []uint64{}
	root := n{
		s:        s,
		alphabet: alphabet,
	}
	queued := []n{root}
	offset := 0
	nodeID := 1
	internalNodes := 1
	for len(queued) > 0 && internalNodes > 0 {
		nodeOffsets = append(nodeOffsets, uint64(offset))

		node := queued[0]
		queued = queued[1:]

		// We can bail early if only remaining nodes are leafs.
		if len(node.alphabet) == 1 {
			// No node here.
			continue
		}
		internalNodes--

		midIdx := (len(node.alphabet) + 1) / 2
		midCh := node.alphabet[midIdx]

		left := []byte{}
		right := []byte{}

		for _, ch := range node.s {
			if ch < midCh {
				left = append(left, ch)
			} else {
				bits.Set(uint64(offset), true)
				right = append(right, ch)
			}
			offset++
		}

		queued = append(queued, n{
			s:        left,
			alphabet: node.alphabet[:midIdx],
		})
		if len(node.alphabet[:midIdx]) > 1 {
			internalNodes++
		}

		queued = append(queued, n{
			s:        right,
			alphabet: node.alphabet[midIdx:],
		})
		if len(node.alphabet[midIdx:]) > 1 {
			internalNodes++
		}

		nodeID++
	}

	// TODO reduce to actual size!
	return bits, nodeOffsets
}

// From math/big/arith.go:

// Length of x in bits.
func bitLen(x uint64) (n int) {
	for ; x >= 0x8000; x >>= 16 {
		n += 16
	}
	if x >= 0x80 {
		x >>= 8
		n += 8
	}
	if x >= 0x8 {
		x >>= 4
		n += 4
	}
	if x >= 0x2 {
		x >>= 2
		n += 2
	}
	if x >= 0x1 {
		n++
	}
	return
}

// log2 computes the integer binary logarithm of x.
// The result is the integer n for which 2^n <= x < 2^(n+1).
// If x == 0, the result is -1.
func log2(x uint64) int {
	return bitLen(x) - 1
}

func uint64SliceAsByteSlice(slice []uint64) []byte {
	// make a new slice header
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&slice))

	// update its capacity and length
	header.Len *= 8
	header.Cap *= 8

	// return it
	return *(*[]byte)(unsafe.Pointer(&header))
}

func byteSliceAsUint64Slice(b []byte) []uint64 {
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&b))

	header.Len /= 8
	header.Cap /= 8

	return *(*[]uint64)(unsafe.Pointer(&header))
}
