// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"encoding/binary"
	"testing"
)

func TestWindowMP3HeaderProbeSkipsLargeID3WithoutRetainingIt(t *testing.T) {
	// MPEG-1 mono, Xing frame count and LAME encoder delay/end padding.
	frame := make([]byte, 512)
	binary.BigEndian.PutUint32(frame, 0xfffb90c0)
	copy(frame[21:], "Xing")
	binary.BigEndian.PutUint32(frame[25:], 1)
	binary.BigEndian.PutUint32(frame[29:], 100)
	copy(frame[33:], "LAME3.100")
	pad := uint32(576<<12 | 700)
	frame[54], frame[55], frame[56] = byte(pad>>16), byte(pad>>8), byte(pad)
	tag := []byte{'I', 'D', '3', 4, 0, 0, 1, 0, 0, 0} // 2 MiB artwork.
	var probe mp3HeaderProbe
	for _, b := range tag {
		probe.Write([]byte{b})
	}
	chunk := make([]byte, 65536)
	for i := 0; i < 32; i++ {
		probe.Write(chunk)
	}
	for _, b := range frame {
		probe.Write([]byte{b})
	}
	if len(probe.data) != len(frame) || probe.offset != 10+2<<20 {
		t.Fatal("ID3 retained or frame lost", len(probe.data), probe.offset)
	}
	g := probe.gapless(10 + 2<<20 + int64(len(frame)))
	if g == nil || g.samples != 100*1152-576-700 || g.untrimmed != 100*1152-576-529 || g.rate != 44100 {
		t.Fatalf("wrong gapless endpoint: %+v", g)
	}
	probe.data[33] = 'X'
	if probe.gapless(1<<23) != nil {
		t.Fatal("unknown encoder delay interpreted")
	}
}

func TestWindowMP3HeaderProbeRejectsIncompleteAndConcatenatedMetadata(t *testing.T) {
	var probe mp3HeaderProbe
	probe.Write([]byte("ID3"))
	if probe.gapless(3) != nil {
		t.Fatal("incomplete metadata accepted")
	}
	frame := make([]byte, 512)
	binary.BigEndian.PutUint32(frame, 0xfffb90c0)
	copy(frame[21:], "Info")
	binary.BigEndian.PutUint32(frame[25:], 3)
	binary.BigEndian.PutUint32(frame[29:], 100)
	binary.BigEndian.PutUint32(frame[33:], 10000)
	copy(frame[37:], "Lavc")
	pad := uint32(576<<12 | 700)
	frame[58], frame[59], frame[60] = byte(pad>>16), byte(pad>>8), byte(pad)
	probe = mp3HeaderProbe{}
	probe.Write(frame)
	if probe.gapless(10000) == nil || probe.gapless(20000) != nil {
		t.Fatal("concatenated frame inventory accepted")
	}
}
