package m3u8

/*
 Part of M3U8 parser & generator library.
 This file defines functions related to playlist generation.

 Copyright 2013-2019 The Project Developers.
 See the AUTHORS and LICENSE files at the top-level directory of this distribution
 and at https://github.com/grafov/m3u8/

 ॐ तारे तुत्तारे तुरे स्व
*/

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// ErrPlaylistFull declares the playlist error.
var ErrPlaylistFull = errors.New("playlist is full")

// ErrPlaylistFull declares the provided playlist is empty
var ErrPlaylistEmpty = errors.New("playlist is empty")

// Set version of the playlist accordingly with section 7
func version(ver *uint8, newver uint8) {
	if *ver < newver {
		*ver = newver
	}
}

func strver(ver uint8) string {
	return strconv.FormatUint(uint64(ver), 10)
}

// NewMasterPlaylist creates a new empty master playlist. Master
// playlist consists of variants.
func NewMasterPlaylist() *MasterPlaylist {
	p := new(MasterPlaylist)
	p.ver = minver
	return p
}

// Append appends a variant to master playlist. This operation does
// reset playlist cache.
func (p *MasterPlaylist) Append(uri string, chunklist *MediaPlaylist, params VariantParams) {
	v := new(Variant)
	v.URI = uri
	v.Chunklist = chunklist
	v.VariantParams = params
	p.Variants = append(p.Variants, v)
	if len(v.Alternatives) > 0 {
		// From section 7:
		// The EXT-X-MEDIA tag and the AUDIO, VIDEO and SUBTITLES attributes of
		// the EXT-X-STREAM-INF tag are backward compatible to protocol version
		// 1, but playback on older clients may not be desirable.  A server MAY
		// consider indicating a EXT-X-VERSION of 4 or higher in the Master
		// Playlist but is not required to do so.
		version(&p.ver, 4) // so it is optional and in theory may be set to ver.1
		// but more tests required
	}
	p.buf.Reset()
}

// ResetCache resetes the playlist' cache.
func (p *MasterPlaylist) ResetCache() {
	p.buf.Reset()
}

// Encode generates the output in M3U8 format.
func (p *MasterPlaylist) Encode() *bytes.Buffer {
	if p.buf.Len() > 0 {
		return &p.buf
	}

	p.buf.WriteString("#EXTM3U\n#EXT-X-VERSION:")
	p.buf.WriteString(strver(p.ver))
	p.buf.WriteRune('\n')

	if p.IndependentSegments() {
		p.buf.WriteString("#EXT-X-INDEPENDENT-SEGMENTS\n")
	}

	// Write any custom master tags
	if p.Custom != nil {
		for _, v := range p.Custom {
			if customBuf := v.Encode(); customBuf != nil {
				p.buf.WriteString(customBuf.String())
				p.buf.WriteRune('\n')
			}
		}
	}

	var languageWritten = make(map[string]bool)
	if p.SessionData != nil {
		for _, sessionData := range p.SessionData {
			// make sure that the same data id and language is not written twice
			if sessionData.Language != "" {
				languageWrittenKey := strings.ToLower(sessionData.DataID) + "-" + strings.ToLower(sessionData.Language)
				if languageWritten[languageWrittenKey] {
					continue
				}
				languageWritten[languageWrittenKey] = true
			}

			p.buf.WriteString("#EXT-X-SESSION-DATA:")
			p.buf.WriteString("DATA-ID=\"")
			p.buf.WriteString(sessionData.DataID)
			p.buf.WriteRune('"')
			if sessionData.Value != "" {
				p.buf.WriteString(",VALUE=\"")
				p.buf.WriteString(sessionData.Value)
				p.buf.WriteRune('"')
			}
			// Each EXT-X-SESSION-DATA tag MUST contain either a VALUE or URI attribute, but not both.
			// In case both are present, default to writing only the VALUE attribute.
			if sessionData.URI != "" && sessionData.Value == "" {
				p.buf.WriteString(",URI=\"")
				p.buf.WriteString(sessionData.URI)
				p.buf.WriteRune('"')
			}
			if sessionData.Language != "" {
				p.buf.WriteString(",LANGUAGE=\"")
				p.buf.WriteString(sessionData.Language)
				p.buf.WriteRune('"')
			}
			p.buf.WriteRune('\n')
		}
	}

	var altsWritten = make(map[string]bool)

	for _, pl := range p.Variants {
		if pl.Alternatives != nil {
			for _, alt := range pl.Alternatives {
				// Make sure that we only write out an alternative once
				altKey := fmt.Sprintf("%s-%s-%s-%s", alt.Type, alt.GroupId, alt.Name, alt.Language)
				if altsWritten[altKey] {
					continue
				}
				altsWritten[altKey] = true

				p.buf.WriteString("#EXT-X-MEDIA:")
				if alt.Type != "" {
					p.buf.WriteString("TYPE=") // Type should not be quoted
					p.buf.WriteString(alt.Type)
				}
				if alt.GroupId != "" {
					p.buf.WriteString(",GROUP-ID=\"")
					p.buf.WriteString(alt.GroupId)
					p.buf.WriteRune('"')
				}
				if alt.Name != "" {
					p.buf.WriteString(",NAME=\"")
					p.buf.WriteString(alt.Name)
					p.buf.WriteRune('"')
				}
				p.buf.WriteString(",DEFAULT=")
				if alt.Default {
					p.buf.WriteString("YES")
				} else {
					p.buf.WriteString("NO")
				}
				if alt.Autoselect != "" {
					p.buf.WriteString(",AUTOSELECT=")
					p.buf.WriteString(alt.Autoselect)
				}
				if alt.Language != "" {
					p.buf.WriteString(",LANGUAGE=\"")
					p.buf.WriteString(alt.Language)
					p.buf.WriteRune('"')
				}
				if alt.Forced != "" {
					p.buf.WriteString(",FORCED=")
					p.buf.WriteString(alt.Forced)
				}
				if alt.Characteristics != "" {
					p.buf.WriteString(",CHARACTERISTICS=\"")
					p.buf.WriteString(alt.Characteristics)
					p.buf.WriteRune('"')
				}
				if alt.Subtitles != "" {
					p.buf.WriteString(",SUBTITLES=\"")
					p.buf.WriteString(alt.Subtitles)
					p.buf.WriteRune('"')
				}
				if alt.URI != "" {
					p.buf.WriteString(",URI=\"")
					p.buf.WriteString(alt.URI)
					p.buf.WriteRune('"')
				}
				if alt.InstreamId != "" {
					p.buf.WriteString(",INSTREAM-ID=\"")
					p.buf.WriteString(alt.InstreamId)
					p.buf.WriteRune('"')
				}
				if alt.Channels != "" {
					p.buf.WriteString(",CHANNELS=\"")
					p.buf.WriteString(alt.Channels)
					p.buf.WriteRune('"')
				}
				p.buf.WriteRune('\n')
			}
		}
		if pl.Iframe {
			p.buf.WriteString("#EXT-X-I-FRAME-STREAM-INF:PROGRAM-ID=")
			p.buf.WriteString(strconv.FormatUint(uint64(pl.ProgramId), 10))
			p.buf.WriteString(",BANDWIDTH=")
			p.buf.WriteString(strconv.FormatUint(uint64(pl.Bandwidth), 10))
			if pl.AverageBandwidth != 0 {
				p.buf.WriteString(",AVERAGE-BANDWIDTH=")
				p.buf.WriteString(strconv.FormatUint(uint64(pl.AverageBandwidth), 10))
			}
			if pl.Codecs != "" {
				p.buf.WriteString(",CODECS=\"")
				p.buf.WriteString(pl.Codecs)
				p.buf.WriteRune('"')
			}
			if pl.Resolution != "" {
				p.buf.WriteString(",RESOLUTION=") // Resolution should not be quoted
				p.buf.WriteString(pl.Resolution)
			}
			if pl.Video != "" {
				p.buf.WriteString(",VIDEO=\"")
				p.buf.WriteString(pl.Video)
				p.buf.WriteRune('"')
			}
			if pl.VideoRange != "" {
				p.buf.WriteString(",VIDEO-RANGE=")
				p.buf.WriteString(pl.VideoRange)
			}
			if pl.HDCPLevel != "" {
				p.buf.WriteString(",HDCP-LEVEL=")
				p.buf.WriteString(pl.HDCPLevel)
			}
			if pl.URI != "" {
				p.buf.WriteString(",URI=\"")
				p.buf.WriteString(pl.URI)
				p.buf.WriteRune('"')
			}
			p.buf.WriteRune('\n')
		} else {
			p.buf.WriteString("#EXT-X-STREAM-INF:PROGRAM-ID=")
			p.buf.WriteString(strconv.FormatUint(uint64(pl.ProgramId), 10))
			p.buf.WriteString(",BANDWIDTH=")
			p.buf.WriteString(strconv.FormatUint(uint64(pl.Bandwidth), 10))
			if pl.AverageBandwidth != 0 {
				p.buf.WriteString(",AVERAGE-BANDWIDTH=")
				p.buf.WriteString(strconv.FormatUint(uint64(pl.AverageBandwidth), 10))
			}
			if pl.Codecs != "" {
				p.buf.WriteString(",CODECS=\"")
				p.buf.WriteString(pl.Codecs)
				p.buf.WriteRune('"')
			}
			if pl.Resolution != "" {
				p.buf.WriteString(",RESOLUTION=") // Resolution should not be quoted
				p.buf.WriteString(pl.Resolution)
			}
			if pl.Audio != "" {
				p.buf.WriteString(",AUDIO=\"")
				p.buf.WriteString(pl.Audio)
				p.buf.WriteRune('"')
			}
			if pl.Video != "" {
				p.buf.WriteString(",VIDEO=\"")
				p.buf.WriteString(pl.Video)
				p.buf.WriteRune('"')
			}
			if pl.Captions != "" {
				p.buf.WriteString(",CLOSED-CAPTIONS=")
				if pl.Captions == "NONE" {
					p.buf.WriteString(pl.Captions) // CC should not be quoted when eq NONE
				} else {
					p.buf.WriteRune('"')
					p.buf.WriteString(pl.Captions)
					p.buf.WriteRune('"')
				}
			}
			if pl.Subtitles != "" {
				p.buf.WriteString(",SUBTITLES=\"")
				p.buf.WriteString(pl.Subtitles)
				p.buf.WriteRune('"')
			}
			if pl.Name != "" {
				p.buf.WriteString(",NAME=\"")
				p.buf.WriteString(pl.Name)
				p.buf.WriteRune('"')
			}
			if pl.FrameRate != 0 {
				p.buf.WriteString(",FRAME-RATE=")
				p.buf.WriteString(strconv.FormatFloat(pl.FrameRate, 'f', 3, 64))
			}
			if pl.VideoRange != "" {
				p.buf.WriteString(",VIDEO-RANGE=")
				p.buf.WriteString(pl.VideoRange)
			}
			if pl.HDCPLevel != "" {
				p.buf.WriteString(",HDCP-LEVEL=")
				p.buf.WriteString(pl.HDCPLevel)
			}

			p.buf.WriteRune('\n')
			p.buf.WriteString(pl.URI)
			if p.Args != "" {
				if strings.Contains(pl.URI, "?") {
					p.buf.WriteRune('&')
				} else {
					p.buf.WriteRune('?')
				}
				p.buf.WriteString(p.Args)
			}
			p.buf.WriteRune('\n')
		}
	}

	return &p.buf
}

// SetCustomTag sets the provided tag on the master playlist for its TagName
func (p *MasterPlaylist) SetCustomTag(tag CustomTag) {
	if p.Custom == nil {
		p.Custom = make(map[string]CustomTag)
	}

	p.Custom[tag.TagName()] = tag
}

// Version returns the current playlist version number
func (p *MasterPlaylist) Version() uint8 {
	return p.ver
}

// SetVersion sets the playlist version number, note the version maybe changed
// automatically by other Set methods.
func (p *MasterPlaylist) SetVersion(ver uint8) {
	p.ver = ver
}

// IndependentSegments returns true if all media samples in a segment can be
// decoded without information from other segments.
func (p *MasterPlaylist) IndependentSegments() bool {
	return p.independentSegments
}

// SetIndependentSegments sets whether all media samples in a segment can be
// decoded without information from other segments.
func (p *MasterPlaylist) SetIndependentSegments(b bool) {
	p.independentSegments = b
}

// String here for compatibility with Stringer interface. For example
// fmt.Printf("%s", sampleMediaList) will encode playist and print its
// string representation.
func (p *MasterPlaylist) String() string {
	return p.Encode().String()
}

// NewMediaPlaylist creates a new media playlist structure. Winsize
// defines how much items will displayed on playlist generation.
// Capacity is total size of a playlist.
func NewMediaPlaylist(winsize uint, capacity uint) (*MediaPlaylist, error) {
	p := new(MediaPlaylist)
	p.ver = minver
	p.capacity = capacity
	if err := p.SetWinSize(winsize); err != nil {
		return nil, err
	}
	p.Segments = make([]*MediaSegment, capacity)
	return p, nil
}

// InsertSegments allows the insertion of one or multiple MediaSegments into the MediaPlaylist.
// seqID is the sequence ID at which the new segments should be inserted to
// This operation does reset playlist cache.
func (p *MediaPlaylist) InsertSegments(segments []*MediaSegment, seqID uint64) error {
	if len(segments) == 0 {
		return ErrPlaylistEmpty
	}

	// Determine the index where the new segments should be inserted
	var insertIndex = 0
	switch {
	case seqID >= uint64(len(p.Segments)):
		insertIndex = len(p.Segments)
	case seqID != 0 && seqID < uint64(len(p.Segments)):
		insertIndex = int(seqID) - 1
	}

	adjustment := uint(len(segments))

	// Shift MediaPlaylist segments in preparation for insertion
	newLength := len(p.Segments) + len(segments)
	if cap(p.Segments) < newLength {
		newSegments := make([]*MediaSegment, newLength)
		copy(newSegments, p.Segments[:insertIndex])
		copy(newSegments[insertIndex+len(segments):], p.Segments[insertIndex:])
		p.Segments = newSegments
	} else {
		p.Segments = p.Segments[:newLength]
		copy(p.Segments[insertIndex+len(segments):], p.Segments[insertIndex:])
	}

	// Insert the segments
	copy(p.Segments[insertIndex:], segments)

	iterator := 1
	// Adjust the sequence IDs of the inserted segments
	for i := insertIndex; i < len(p.Segments[:insertIndex])+len(segments); i++ {
		p.Segments[i].SeqId = uint64(insertIndex + iterator)
		iterator++
	}

	// Adjust the sequence IDs of the following segments
	for i := insertIndex + len(segments); i < len(p.Segments); i++ {
		if p.Segments[i] != nil {
			p.Segments[i].SeqId += uint64(adjustment)
		}
	}

	p.count += adjustment
	p.capacity += adjustment
	p.tail = p.count

	p.buf.Reset()
	return nil
}

// SetMediaSegments takes in []*MediaSegment and sets the playlist segments to it
// This operation does reset playlist cache
func (p *MediaPlaylist) SetMediaSegments(segments []*MediaSegment) {
	p.Segments = segments

	p.count = uint(len(segments))
	p.tail = p.count
	p.capacity = p.count
	p.buf.Reset()
}

// last returns the previously written segment's index
func (p *MediaPlaylist) last() uint {
	if p.tail == 0 {
		return p.capacity - 1
	}
	return p.tail - 1
}

// Remove current segment from the head of chunk slice form a media playlist. Useful for sliding playlists.
// This operation does reset playlist cache.
func (p *MediaPlaylist) Remove() (err error) {
	if p.count == 0 {
		return errors.New("playlist is empty")
	}
	p.head = (p.head + 1) % p.capacity
	p.count--
	if !p.Closed {
		p.SeqNo++
	}
	p.buf.Reset()
	return nil
}

// Append general chunk to the tail of chunk slice for a media playlist.
// This operation does reset playlist cache.
func (p *MediaPlaylist) Append(uri string, duration float64, title string) error {
	seg := new(MediaSegment)
	seg.URI = uri
	seg.Duration = duration
	seg.Title = title
	return p.AppendSegment(seg)
}

// AppendSegment appends a MediaSegment to the tail of chunk slice for
// a media playlist.  This operation does reset playlist cache.
func (p *MediaPlaylist) AppendSegment(seg *MediaSegment) error {
	if p.head == p.tail && p.count > 0 {
		return ErrPlaylistFull
	}
	seg.SeqId = p.SeqNo
	if p.count > 0 {
		seg.SeqId = p.Segments[(p.capacity+p.tail-1)%p.capacity].SeqId + 1
	}
	p.Segments[p.tail] = seg
	p.tail = (p.tail + 1) % p.capacity
	p.count++
	if p.TargetDuration < seg.Duration {
		p.TargetDuration = math.Ceil(seg.Duration)
	}
	p.buf.Reset()
	return nil
}

// Slide combines two operations: firstly it removes one chunk from
// the head of chunk slice and move pointer to next chunk. Secondly it
// appends one chunk to the tail of chunk slice. Useful for sliding
// playlists.  This operation does reset cache.
func (p *MediaPlaylist) Slide(uri string, duration float64, title string) {
	if !p.Closed && p.count >= p.winsize {
		p.Remove()
	}
	p.Append(uri, duration, title)
}

// ResetCache resets playlist cache. Next called Encode() will
// regenerate playlist from the chunk slice.
func (p *MediaPlaylist) ResetCache() {
	p.buf.Reset()
}

// IndependentSegments returns true if all media samples in a segment can be
// decoded without information from other segments.
func (p *MediaPlaylist) IndependentSegments() bool {
	return p.independentSegments
}

// SetIndependentSegments sets whether all media samples in a segment can be
// decoded without information from other segments.
func (p *MediaPlaylist) SetIndependentSegments(b bool) {
	p.independentSegments = b
}

// Encode generates output in M3U8 format. Marshal `winsize` elements
// from bottom of the `segments` queue.
func (p *MediaPlaylist) Encode() *bytes.Buffer {
	if p.buf.Len() > 0 {
		return &p.buf
	}

	p.buf.WriteString("#EXTM3U\n#EXT-X-VERSION:")
	p.buf.WriteString(strver(p.ver))
	p.buf.WriteRune('\n')

	if p.IndependentSegments() {
		p.buf.WriteString("#EXT-X-INDEPENDENT-SEGMENTS\n")
	}

	// Write any custom master tags
	if p.Custom != nil {
		for _, v := range p.Custom {
			if customBuf := v.Encode(); customBuf != nil {
				p.buf.WriteString(customBuf.String())
				p.buf.WriteRune('\n')
			}
		}
	}

	// default key (workaround for Widevine)
	if p.Key != nil {
		p.buf.WriteString("#EXT-X-KEY:")
		p.buf.WriteString("METHOD=")
		p.buf.WriteString(p.Key.Method)
		if p.Key.Method != "NONE" {
			p.buf.WriteString(",URI=\"")
			p.buf.WriteString(p.Key.URI)
			p.buf.WriteRune('"')
			if p.Key.IV != "" {
				p.buf.WriteString(",IV=")
				p.buf.WriteString(p.Key.IV)
			}
			if p.Key.Keyformat != "" {
				p.buf.WriteString(",KEYFORMAT=\"")
				p.buf.WriteString(p.Key.Keyformat)
				p.buf.WriteRune('"')
			}
			if p.Key.Keyformatversions != "" {
				p.buf.WriteString(",KEYFORMATVERSIONS=\"")
				p.buf.WriteString(p.Key.Keyformatversions)
				p.buf.WriteRune('"')
			}
		}
		p.buf.WriteRune('\n')
	}
	if p.Map != nil {
		p.buf.WriteString("#EXT-X-MAP:")
		p.buf.WriteString("URI=\"")
		p.buf.WriteString(p.Map.URI)
		p.buf.WriteRune('"')
		if p.Map.Limit > 0 {
			p.buf.WriteString(",BYTERANGE=")
			p.buf.WriteString(strconv.FormatInt(p.Map.Limit, 10))
			p.buf.WriteRune('@')
			p.buf.WriteString(strconv.FormatInt(p.Map.Offset, 10))
		}
		p.buf.WriteRune('\n')
	}
	if p.MediaType > 0 {
		p.buf.WriteString("#EXT-X-PLAYLIST-TYPE:")
		switch p.MediaType {
		case EVENT:
			p.buf.WriteString("EVENT\n")
			p.buf.WriteString("#EXT-X-ALLOW-CACHE:NO\n")
		case VOD:
			p.buf.WriteString("VOD\n")
		}
	}
	p.buf.WriteString("#EXT-X-MEDIA-SEQUENCE:")
	p.buf.WriteString(strconv.FormatUint(p.SeqNo, 10))
	p.buf.WriteRune('\n')
	p.buf.WriteString("#EXT-X-TARGETDURATION:")
	p.buf.WriteString(strconv.FormatInt(int64(math.Ceil(p.TargetDuration)), 10)) // due section 3.4.2 of M3U8 specs EXT-X-TARGETDURATION must be integer
	p.buf.WriteRune('\n')
	if p.StartTime > 0.0 {
		p.buf.WriteString("#EXT-X-START:TIME-OFFSET=")
		p.buf.WriteString(strconv.FormatFloat(p.StartTime, 'f', -1, 64))
		if p.StartTimePrecise {
			p.buf.WriteString(",PRECISE=YES")
		}
		p.buf.WriteRune('\n')
	}
	if p.DiscontinuitySeq != 0 {
		p.buf.WriteString("#EXT-X-DISCONTINUITY-SEQUENCE:")
		p.buf.WriteString(strconv.FormatUint(uint64(p.DiscontinuitySeq), 10))
		p.buf.WriteRune('\n')
	}
	if p.Iframe {
		p.buf.WriteString("#EXT-X-I-FRAMES-ONLY\n")
	}
	// Widevine tags
	if p.WV != nil {
		if p.WV.AudioChannels != 0 {
			p.buf.WriteString("#WV-AUDIO-CHANNELS ")
			p.buf.WriteString(strconv.FormatUint(uint64(p.WV.AudioChannels), 10))
			p.buf.WriteRune('\n')
		}
		if p.WV.AudioFormat != 0 {
			p.buf.WriteString("#WV-AUDIO-FORMAT ")
			p.buf.WriteString(strconv.FormatUint(uint64(p.WV.AudioFormat), 10))
			p.buf.WriteRune('\n')
		}
		if p.WV.AudioProfileIDC != 0 {
			p.buf.WriteString("#WV-AUDIO-PROFILE-IDC ")
			p.buf.WriteString(strconv.FormatUint(uint64(p.WV.AudioProfileIDC), 10))
			p.buf.WriteRune('\n')
		}
		if p.WV.AudioSampleSize != 0 {
			p.buf.WriteString("#WV-AUDIO-SAMPLE-SIZE ")
			p.buf.WriteString(strconv.FormatUint(uint64(p.WV.AudioSampleSize), 10))
			p.buf.WriteRune('\n')
		}
		if p.WV.AudioSamplingFrequency != 0 {
			p.buf.WriteString("#WV-AUDIO-SAMPLING-FREQUENCY ")
			p.buf.WriteString(strconv.FormatUint(uint64(p.WV.AudioSamplingFrequency), 10))
			p.buf.WriteRune('\n')
		}
		if p.WV.CypherVersion != "" {
			p.buf.WriteString("#WV-CYPHER-VERSION ")
			p.buf.WriteString(p.WV.CypherVersion)
			p.buf.WriteRune('\n')
		}
		if p.WV.ECM != "" {
			p.buf.WriteString("#WV-ECM ")
			p.buf.WriteString(p.WV.ECM)
			p.buf.WriteRune('\n')
		}
		if p.WV.VideoFormat != 0 {
			p.buf.WriteString("#WV-VIDEO-FORMAT ")
			p.buf.WriteString(strconv.FormatUint(uint64(p.WV.VideoFormat), 10))
			p.buf.WriteRune('\n')
		}
		if p.WV.VideoFrameRate != 0 {
			p.buf.WriteString("#WV-VIDEO-FRAME-RATE ")
			p.buf.WriteString(strconv.FormatUint(uint64(p.WV.VideoFrameRate), 10))
			p.buf.WriteRune('\n')
		}
		if p.WV.VideoLevelIDC != 0 {
			p.buf.WriteString("#WV-VIDEO-LEVEL-IDC")
			p.buf.WriteString(strconv.FormatUint(uint64(p.WV.VideoLevelIDC), 10))
			p.buf.WriteRune('\n')
		}
		if p.WV.VideoProfileIDC != 0 {
			p.buf.WriteString("#WV-VIDEO-PROFILE-IDC ")
			p.buf.WriteString(strconv.FormatUint(uint64(p.WV.VideoProfileIDC), 10))
			p.buf.WriteRune('\n')
		}
		if p.WV.VideoResolution != "" {
			p.buf.WriteString("#WV-VIDEO-RESOLUTION ")
			p.buf.WriteString(p.WV.VideoResolution)
			p.buf.WriteRune('\n')
		}
		if p.WV.VideoSAR != "" {
			p.buf.WriteString("#WV-VIDEO-SAR ")
			p.buf.WriteString(p.WV.VideoSAR)
			p.buf.WriteRune('\n')
		}
	}

	var (
		seg           *MediaSegment
		durationCache = make(map[float64]string)
	)

	head := p.head
	count := p.count
	for i := uint(0); (i < p.winsize || p.winsize == 0) && count > 0; count-- {
		seg = p.Segments[head]
		head = (head + 1) % p.capacity
		if seg == nil { // protection from badly filled chunklists
			continue
		}
		if p.winsize > 0 { // skip for VOD playlists, where winsize = 0
			i++
		}
		if seg.SCTE != nil {
			switch seg.SCTE.Syntax {
			case SCTE35_67_2014:
				p.buf.WriteString("#EXT-SCTE35:")
				p.buf.WriteString("CUE=\"")
				p.buf.WriteString(seg.SCTE.Cue)
				p.buf.WriteRune('"')
				if seg.SCTE.ID != "" {
					p.buf.WriteString(",ID=\"")
					p.buf.WriteString(seg.SCTE.ID)
					p.buf.WriteRune('"')
				}
				if seg.SCTE.Time != 0 {
					p.buf.WriteString(",TIME=")
					p.buf.WriteString(strconv.FormatFloat(seg.SCTE.Time, 'f', -1, 64))
				}
				p.buf.WriteRune('\n')
			case SCTE35_OATCLS:
				switch seg.SCTE.CueType {
				case SCTE35Cue_Start:
					if seg.SCTE.Cue != "" {
						p.buf.WriteString("#EXT-OATCLS-SCTE35:")
						p.buf.WriteString(seg.SCTE.Cue)
						p.buf.WriteRune('\n')
					}
					p.buf.WriteString("#EXT-X-CUE-OUT:")
					p.buf.WriteString(strconv.FormatFloat(seg.SCTE.Time, 'f', -1, 64))
					p.buf.WriteRune('\n')
				case SCTE35Cue_Mid:
					p.buf.WriteString("#EXT-X-CUE-OUT-CONT:")
					p.buf.WriteString("ElapsedTime=")
					p.buf.WriteString(strconv.FormatFloat(seg.SCTE.Elapsed, 'f', -1, 64))
					p.buf.WriteString(",Duration=")
					p.buf.WriteString(strconv.FormatFloat(seg.SCTE.Time, 'f', -1, 64))
					p.buf.WriteString(",SCTE35=")
					p.buf.WriteString(seg.SCTE.Cue)
					p.buf.WriteRune('\n')
				case SCTE35Cue_End:
					p.buf.WriteString("#EXT-X-CUE-IN")
					p.buf.WriteRune('\n')
				}
			}
		}
		// check for key change
		if seg.Key != nil && p.Key != seg.Key {
			p.buf.WriteString("#EXT-X-KEY:")
			p.buf.WriteString("METHOD=")
			p.buf.WriteString(seg.Key.Method)
			if seg.Key.Method != "NONE" {
				p.buf.WriteString(",URI=\"")
				p.buf.WriteString(seg.Key.URI)
				p.buf.WriteRune('"')
				if seg.Key.IV != "" {
					p.buf.WriteString(",IV=")
					p.buf.WriteString(seg.Key.IV)
				}
				if seg.Key.Keyformat != "" {
					p.buf.WriteString(",KEYFORMAT=\"")
					p.buf.WriteString(seg.Key.Keyformat)
					p.buf.WriteRune('"')
				}
				if seg.Key.Keyformatversions != "" {
					p.buf.WriteString(",KEYFORMATVERSIONS=\"")
					p.buf.WriteString(seg.Key.Keyformatversions)
					p.buf.WriteRune('"')
				}
			}
			p.buf.WriteRune('\n')
		}
		if len(seg.DateRange) > 0 {
			for _, dr := range seg.DateRange {
				p.buf.WriteString("#EXT-X-DATERANGE:")
				p.buf.WriteString("ID=\"")
				p.buf.WriteString(dr.ID)
				p.buf.WriteRune('"')
				if dr.Class != "" {
					p.buf.WriteString(",CLASS=\"")
					p.buf.WriteString(dr.Class)
					p.buf.WriteRune('"')
				}
				if !dr.StartDate.IsZero() {
					p.buf.WriteString(",START-DATE=\"")
					p.buf.WriteString(dr.StartDate.Format(DATETIME))
					p.buf.WriteRune('"')
				}
				if !dr.EndDate.IsZero() {
					p.buf.WriteString(",END-DATE=\"")
					p.buf.WriteString(dr.EndDate.Format(DATETIME))
					p.buf.WriteRune('"')
				}
				if dr.Duration > 0 {
					p.buf.WriteString(",DURATION=")
					p.buf.WriteString(strconv.FormatFloat(dr.Duration, 'f', -1, 64))
				}
				if dr.PlannedDuration > 0 {
					p.buf.WriteString(",PLANNED-DURATION=")
					p.buf.WriteString(strconv.FormatFloat(dr.PlannedDuration, 'f', -1, 64))
				}
				if dr.SCTE35Cmd != "" {
					p.buf.WriteString(",SCTE35-CMD=")
					p.buf.WriteString(dr.SCTE35Cmd)
				}
				if dr.SCTE35In != "" {
					p.buf.WriteString(",SCTE35-IN=")
					p.buf.WriteString(dr.SCTE35In)
				}
				if dr.SCTE35Out != "" {
					p.buf.WriteString(",SCTE35-OUT=")
					p.buf.WriteString(dr.SCTE35Out)
				}
				if dr.EndOnNext != "" {
					p.buf.WriteString(",END-ON-NEXT=\"")
					p.buf.WriteString(dr.EndOnNext)
					p.buf.WriteRune('"')
				}
				if dr.XResumeOfsset > 0 {
					p.buf.WriteString(",X-RESUME-OFFSET=")
					p.buf.WriteString(strconv.FormatFloat(dr.Duration, 'f', -1, 64))
				}
				if dr.XPlayoutLimit > 0 {
					p.buf.WriteString(",X-PLAYOUT-LIMIT=")
					p.buf.WriteString(strconv.FormatFloat(dr.Duration, 'f', -1, 64))
				}
				if dr.XSnap != "" {
					p.buf.WriteString(",X-SNAP=\"")
					p.buf.WriteString(dr.XSnap)
					p.buf.WriteRune('"')
				}
				if dr.XRestrict != "" {
					p.buf.WriteString(",X-RESTRICT=\"")
					p.buf.WriteString(dr.XRestrict)
					p.buf.WriteRune('"')
				}
				if dr.XAssetURI != "" {
					p.buf.WriteString(",X-ASSET-URI=\"")
					p.buf.WriteString(dr.XAssetURI)
					p.buf.WriteRune('"')
				}
				if dr.XAssetList != "" {
					p.buf.WriteString(",X-ASSET-LIST=\"")
					p.buf.WriteString(dr.XAssetList)
					p.buf.WriteRune('"')
				}
				for k, v := range dr.X {
					p.buf.WriteString(",")
					p.buf.WriteString(k)
					p.buf.WriteString("=\"")
					p.buf.WriteString(v)
					p.buf.WriteRune('"')
				}
				p.buf.WriteString("\n")
			}
		}
		if seg.Discontinuity {
			p.buf.WriteString("#EXT-X-DISCONTINUITY\n")
		}
		if seg.Gap {
			p.buf.WriteString("#EXT-X-GAP\n")
		}
		// ignore segment Map if default playlist Map is present
		if p.Map == nil && seg.Map != nil {
			p.buf.WriteString("#EXT-X-MAP:")
			p.buf.WriteString("URI=\"")
			p.buf.WriteString(seg.Map.URI)
			p.buf.WriteRune('"')
			if seg.Map.Limit > 0 {
				p.buf.WriteString(",BYTERANGE=")
				p.buf.WriteString(strconv.FormatInt(seg.Map.Limit, 10))
				p.buf.WriteRune('@')
				p.buf.WriteString(strconv.FormatInt(seg.Map.Offset, 10))
			}
			p.buf.WriteRune('\n')
		}
		if !seg.ProgramDateTime.IsZero() {
			p.buf.WriteString("#EXT-X-PROGRAM-DATE-TIME:")
			p.buf.WriteString(seg.ProgramDateTime.Format(DATETIME))
			p.buf.WriteRune('\n')
		}
		if seg.Limit > 0 {
			p.buf.WriteString("#EXT-X-BYTERANGE:")
			p.buf.WriteString(strconv.FormatInt(seg.Limit, 10))
			p.buf.WriteRune('@')
			p.buf.WriteString(strconv.FormatInt(seg.Offset, 10))
			p.buf.WriteRune('\n')
		}

		// Add Custom Segment Tags here
		if seg.Custom != nil {
			for _, v := range seg.Custom {
				if customBuf := v.Encode(); customBuf != nil {
					p.buf.WriteString(customBuf.String())
					p.buf.WriteRune('\n')
				}
			}
		}

		p.buf.WriteString("#EXTINF:")
		if str, ok := durationCache[seg.Duration]; ok {
			p.buf.WriteString(str)
		} else {
			if p.durationAsInt {
				// Old Android players has problems with non integer Duration.
				durationCache[seg.Duration] = strconv.FormatInt(int64(math.Ceil(seg.Duration)), 10)
			} else {
				// Wowza Mediaserver and some others prefer floats.
				durationCache[seg.Duration] = strconv.FormatFloat(seg.Duration, 'f', 3, 32)
			}
			p.buf.WriteString(durationCache[seg.Duration])
		}
		p.buf.WriteRune(',')
		p.buf.WriteString(seg.Title)
		p.buf.WriteRune('\n')
		p.buf.WriteString(seg.URI)
		if p.Args != "" {
			p.buf.WriteRune('?')
			p.buf.WriteString(p.Args)
		}
		p.buf.WriteRune('\n')
	}
	if p.Closed {
		p.buf.WriteString("#EXT-X-ENDLIST\n")
	}
	return &p.buf
}

// String here for compatibility with Stringer interface For example
// fmt.Printf("%s", sampleMediaList) will encode playist and print its
// string representation.
func (p *MediaPlaylist) String() string {
	return p.Encode().String()
}

// DurationAsInt represents the duration as the integer in encoded playlist.
func (p *MediaPlaylist) DurationAsInt(yes bool) {
	if yes {
		// duration must be integers if protocol version is less than 3
		version(&p.ver, 3)
	}
	p.durationAsInt = yes
}

// Count tells us the number of items that are currently in the media
// playlist.
func (p *MediaPlaylist) Count() uint {
	return p.count
}

// Close sliding playlist and make them fixed.
func (p *MediaPlaylist) Close() {
	if p.buf.Len() > 0 {
		p.buf.WriteString("#EXT-X-ENDLIST\n")
	}
	p.Closed = true
}

// SetDefaultKey sets encryption key appeared once in header of the
// playlist (pointer to MediaPlaylist.Key). It useful when keys not
// changed during playback.  Set tag for the whole list.
func (p *MediaPlaylist) SetDefaultKey(method, uri, iv, keyformat, keyformatversions string) error {
	// A Media Playlist MUST indicate a EXT-X-VERSION of 5 or higher if it
	// contains:
	//   - The KEYFORMAT and KEYFORMATVERSIONS attributes of the EXT-X-KEY tag.
	if keyformat != "" || keyformatversions != "" {
		version(&p.ver, 5)
	}
	p.Key = &Key{method, uri, iv, keyformat, keyformatversions}

	return nil
}

// SetDefaultMap sets default Media Initialization Section values for
// playlist (pointer to MediaPlaylist.Map). Set EXT-X-MAP tag for the
// whole playlist.
func (p *MediaPlaylist) SetDefaultMap(uri string, limit, offset int64) {
	version(&p.ver, 5) // due section 4
	p.Map = &Map{uri, limit, offset}
}

// SetIframeOnly marks medialist as consists of only I-frames (Intra
// frames).  Set tag for the whole list.
func (p *MediaPlaylist) SetIframeOnly() {
	version(&p.ver, 4) // due section 4.3.3
	p.Iframe = true
}

// SetKey sets encryption key for the current segment of media playlist
// (pointer to Segment.Key).
func (p *MediaPlaylist) SetKey(method, uri, iv, keyformat, keyformatversions string) error {
	if p.count == 0 {
		return errors.New("playlist is empty")
	}

	// A Media Playlist MUST indicate a EXT-X-VERSION of 5 or higher if it
	// contains:
	//   - The KEYFORMAT and KEYFORMATVERSIONS attributes of the EXT-X-KEY tag.
	if keyformat != "" || keyformatversions != "" {
		version(&p.ver, 5)
	}

	p.Segments[p.last()].Key = &Key{method, uri, iv, keyformat, keyformatversions}
	return nil
}

// SetMap sets map for the current segment of media playlist (pointer
// to Segment.Map).
func (p *MediaPlaylist) SetMap(uri string, limit, offset int64) error {
	if p.count == 0 {
		return errors.New("playlist is empty")
	}
	version(&p.ver, 5) // due section 4
	p.Segments[p.last()].Map = &Map{uri, limit, offset}
	return nil
}

// SetRange sets limit and offset for the current media segment
// (EXT-X-BYTERANGE support for protocol version 4).
func (p *MediaPlaylist) SetRange(limit, offset int64) error {
	if p.count == 0 {
		return errors.New("playlist is empty")
	}
	version(&p.ver, 4) // due section 3.4.1
	p.Segments[p.last()].Limit = limit
	p.Segments[p.last()].Offset = offset
	return nil
}

// SetSCTE sets the SCTE cue format for the current media segment.
//
// Deprecated: Use SetSCTE35 instead.
func (p *MediaPlaylist) SetSCTE(cue string, id string, time float64) error {
	return p.SetSCTE35(&SCTE{Syntax: SCTE35_67_2014, Cue: cue, ID: id, Time: time})
}

// SetSCTE35 sets the SCTE cue format for the current media segment
func (p *MediaPlaylist) SetSCTE35(scte35 *SCTE) error {
	if p.count == 0 {
		return errors.New("playlist is empty")
	}
	p.Segments[p.last()].SCTE = scte35
	return nil
}

// SetDateRange sets DateRange to the current media segment
func (p *MediaPlaylist) SetDateRange(drs []*DateRange) error {
	if p.count == 0 {
		return errors.New("playlist is empty")
	}
	for _, dr := range drs {
		if dr.ID == "" {
			return errors.New("DateRange ID")
		}
	}
	p.Segments[p.last()].DateRange = drs
	return nil
}

// AppendDateRange appends DateRange to the current media segment
func (p *MediaPlaylist) AppendDateRange(dr *DateRange) error {
	if p.count == 0 {
		return errors.New("playlist is empty")
	}
	if dr.ID == "" {
		return errors.New("DateRange ID")
	}
	p.Segments[p.last()].DateRange = append(p.Segments[p.last()].DateRange, dr)
	return nil
}

// SetDiscontinuity sets discontinuity flag for the current media
// segment. EXT-X-DISCONTINUITY indicates an encoding discontinuity
// between the media segment that follows it and the one that preceded
// it (i.e. file format, number and type of tracks, encoding
// parameters, encoding sequence, timestamp sequence).
func (p *MediaPlaylist) SetDiscontinuity() error {
	if p.count == 0 {
		return errors.New("playlist is empty")
	}
	p.Segments[p.last()].Discontinuity = true
	return nil
}

// SetGap sets gap flag for the current media
// segment. EXT-X-GAP indicates that the segment URI to which it applies
// does not contain media data and SHOULD NOT be loaded by clients/
func (p *MediaPlaylist) SetGap() error {
	if p.count == 0 {
		return errors.New("playlist is empty")
	}
	p.Segments[p.last()].Gap = true
	return nil
}

// SetProgramDateTime sets program date and time for the current media
// segment. EXT-X-PROGRAM-DATE-TIME tag associates the first sample of
// a media segment with an absolute date and/or time. It applies only
// to the current media segment.  Date/time format is
// YYYY-MM-DDThh:mm:ssZ (ISO8601) and includes time zone.
func (p *MediaPlaylist) SetProgramDateTime(value time.Time) error {
	if p.count == 0 {
		return errors.New("playlist is empty")
	}
	p.Segments[p.last()].ProgramDateTime = value
	return nil
}

// SetCustomTag sets the provided tag on the media playlist for its
// TagName.
func (p *MediaPlaylist) SetCustomTag(tag CustomTag) {
	if p.Custom == nil {
		p.Custom = make(map[string]CustomTag)
	}

	p.Custom[tag.TagName()] = tag
}

// SetCustomSegmentTag sets the provided tag on the current media
// segment for its TagName.
func (p *MediaPlaylist) SetCustomSegmentTag(tag CustomTag) error {
	if p.count == 0 {
		return errors.New("playlist is empty")
	}

	last := p.Segments[p.last()]

	if last.Custom == nil {
		last.Custom = make(map[string]CustomTag)
	}

	last.Custom[tag.TagName()] = tag

	return nil
}

// Version returns the current playlist version number
func (p *MediaPlaylist) Version() uint8 {
	return p.ver
}

// SetVersion sets the playlist version number, note the version maybe changed
// automatically by other Set methods.
func (p *MediaPlaylist) SetVersion(ver uint8) {
	p.ver = ver
}

// WinSize returns the playlist's window size.
func (p *MediaPlaylist) WinSize() uint {
	return p.winsize
}

// SetWinSize overwrites the playlist's window size.
func (p *MediaPlaylist) SetWinSize(winsize uint) error {
	if winsize > p.capacity {
		return errors.New("capacity must be greater than winsize or equal")
	}
	p.winsize = winsize
	return nil
}
