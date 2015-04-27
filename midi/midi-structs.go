package midi

/*
This file contains data structures used by the midi package.
*/

const (
	msbMask       = 1 << 7
	sevenBitMask  = 0x7F
	highOrderMask = 0xF0
	lowOrderMasl  = 0x0F

	// The following byte constants represent the set of Channel Voice
	// events seen in a MIDI message TrackEvent.
	NoteOffEvent          = 0x80
	NoteOnEvent           = 0x90
	PolyphonicKeyPressure = 0xA0
	ControlChange         = 0xB0
	ProgramChange         = 0xC0
	ChannelPressure       = 0xD0
	PitchWheelChange      = 0xE0
)

var (
	headerChunk = [4]byte{'M', 'T', 'h', 'd'}
	trackChunk  = [4]byte{'M', 'T', 'r', 'k'}
)

/*
Midi represents a MIDI file as defined by the MIDI file spec:
http://goo.gl/rlEN0H
*/
type Midi struct {
	*HeaderChunk
	TrackChunks []TrackChunk
}

/*
Chunks are the basic building block of Midi files. All Chunks contain a 4
character type and a 32-bit length which is the number of bytes contained in the
data of the chunk. The data always immediately follows the chunk. e.g.
    <Chunk<type><length>><data of length><Chunk<type><length>><data of length>
*/
type Chunk struct {
	Type   [4]byte
	Length uint32
}

/*
A HeaderChunk defines the first type of chunk that should be encountered in
every MIDI file. It contains basic information about the data in the file. The
format is:
    <Header Chunk> = <chunk type><length><format><ntrks><division>
The data section contains three 16-bit words, stored most-significant byte
first. The first word, <format>, specifies the overall organisation of the file.
The next word, <ntrks>, is the number of track chunks in the file. It will
always be 1 for a format 0 file. The third word, <division>, specifies the
meaning of the delta-times.
*/
type HeaderChunk struct {
	*Chunk
	Format   uint16
	Ntrks    uint16
	Division uint16
}

/*
A TrackChunk contains the data for the MIDI file. In most cases, there is a
single track. The contains one or more TrackEvent objects that define the sound
events that make up the song.
*/
type TrackChunk struct {
	*Chunk
	TrackEvents []TrackEvent
}

/*
A TrackEvent contains 'events' that occur over the course of the MIDI file.
The syntax of an MTrk event is very simple:
    <TrackEvent> = <delta-time><MidiEvent>
<delta-time> is stored as a variable-length quantity. It represents the amount
of time before the following event. Delta-times are always present, even when 0.
*/
type TrackEvent struct {
	DeltaTime int
	Data      []byte
}
