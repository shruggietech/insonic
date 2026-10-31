// SPDX-License-Identifier: Apache-2.0
package processing

import "encoding/binary"

// mp3HeaderProbe retains only the first MPEG frame after ID3 tags, even when
// artwork makes an ID3 tag larger than the verification chunk. It observes
// accepted bytes during the initial digest pass and never rewrites them.
type mp3HeaderProbe struct {
	data   []byte
	skip   int64
	offset int64
	header bool
}

func (p *mp3HeaderProbe) Write(data []byte) {
	for len(data) > 0 {
		if p.skip > 0 {
			n := min(int64(len(data)), p.skip)
			p.skip -= n
			p.offset += n
			data = data[n:]
			continue
		}
		limit := 10
		if p.header {
			limit = 4096
		}
		n := min(len(data), limit-len(p.data))
		p.data = append(p.data, data[:n]...)
		data = data[n:]
		if len(p.data) < limit {
			return
		}
		if p.header {
			return
		}
		if string(p.data[:3]) == "ID3" && p.data[3] < 5 && p.data[3] > 1 && p.data[6]&0x80 == 0 && p.data[7]&0x80 == 0 && p.data[8]&0x80 == 0 && p.data[9]&0x80 == 0 {
			p.skip = int64(p.data[6])<<21 | int64(p.data[7])<<14 | int64(p.data[8])<<7 | int64(p.data[9])
			if p.data[3] == 4 && p.data[5]&0x10 != 0 {
				p.skip += 10
			}
			p.offset += 10
			p.data = p.data[:0]
			continue
		}
		p.header = true
	}
}

type mp3Gapless struct{ samples, untrimmed, rate int64 }

func (p *mp3HeaderProbe) gapless(size int64) *mp3Gapless {
	b := p.data
	if len(b) < 4 {
		return nil
	}
	h := binary.BigEndian.Uint32(b)
	version := (h >> 19) & 3
	if h&0xffe00000 != 0xffe00000 || version == 1 || (h>>17)&3 != 1 || (h>>12)&15 == 0 || (h>>12)&15 == 15 || (h>>10)&3 == 3 {
		return nil
	}
	rate := []int64{44100, 48000, 32000}[(h>>10)&3]
	spf, side := int64(1152), 32
	mono := (h>>6)&3 == 3
	if mono {
		side = 17
	}
	if version != 3 {
		spf = 576
		rate /= 2
		side = 17
		if mono {
			side = 9
		}
		if version == 0 {
			rate /= 2
		}
	}
	at := 4 + side
	if len(b) < at+8 || string(b[at:at+4]) != "Xing" && string(b[at:at+4]) != "Info" {
		return nil
	}
	flags := binary.BigEndian.Uint32(b[at+4:])
	at += 8
	if flags&1 == 0 || len(b) < at+4 {
		return nil
	}
	frames := int64(binary.BigEndian.Uint32(b[at:]))
	at += 4
	if flags&2 != 0 {
		if len(b) < at+4 {
			return nil
		}
		declared := int64(binary.BigEndian.Uint32(b[at:]))
		at += 4
		// Match the seekable demuxer's concatenated-file test. Such a file
		// cannot use its first segment's gapless frame inventory.
		actual := size - p.offset - 4
		if declared > 0 && actual > declared && actual-declared > declared/16 {
			return nil
		}
	}
	if flags&4 != 0 {
		at += 100
	}
	if flags&8 != 0 {
		at += 4
	}
	if len(b) < at+24 {
		return nil
	}
	encoder := string(b[at : at+4])
	if encoder != "LAME" && encoder != "Lavf" && encoder != "Lavc" {
		return nil
	}
	pad := uint32(b[at+21])<<16 | uint32(b[at+22])<<8 | uint32(b[at+23])
	start, end := int64(pad>>12), int64(pad&4095)
	if frames == 0 || end < 529 || frames*spf < start+end {
		return nil
	}
	return &mp3Gapless{samples: frames*spf - start - end, untrimmed: frames*spf - start - 529, rate: rate}
}
