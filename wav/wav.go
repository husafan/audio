/*
wav implements utilities for reading and writing WAV sound files. More
information on the WAV sound file format can be found here:

WAV File Format implemented within: http://goo.gl/Wi3NNU.
*/
package wav

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	ChannelError           = "expected %v channels; found %v."
	Data                   = "data"
	DataError              = "invalid data chunk ID of %s; should be 'data'"
	Fmt                    = "fmt "
	FmtError               = "invalid fmt chunk ID of %s; should be 'fmt '."
	Riff                   = "RIFF"
	RiffError              = "invalid initial chunk ID of %s; should be 'RIFF'"
	SampleError            = "expected %v bytes per sample but only found %v"
	Wave                   = "WAVE"
	WaveError              = "invalid format of %s; should be 'WAVE'"
	FormatChunkError       = "invalid format chunk: %s."
	RiffSizeOffset   int64 = 4
	DataSizeOffset   int64 = 40
	DataOffset       int64 = 44
)

/*
defaultRiffHeader conains an ID of 'RIFF' and a format of 'WAVE'. It contains a
default size of 36. Every time a sample is added, the size should be incremented
by the number BytesPerSample * NumOfChannels.
*/
var defaultRiffHeader *RiffHeader = &RiffHeader{
	&SubChunk{
		Id:   uint32(1380533830),
		Size: uint32(36),
	},
	uint32(1463899717),
}

/*
defaultDataChunk has an ID of 'data' and a size of 0 with no samples to start
with.
*/
var defaultDataChunk *DataChunk = &DataChunk{
	SubChunk: &SubChunk{
		Id: uint32(1684108385),
	},
}

/*
SubChunk defines the common properties of all WAV file chunks. It includes an ID
and a size in bytes. The ID is big-endian while the size is little-endian.
*/
type SubChunk struct {
	Id   uint32
	Size uint32
}

/*
RiffHeader chunk defines the first chunk in a well-formed WAV file. The ID of
the chunk is always the four ASCII characters "RIFF". It is followed by the size
of the entire WAV file. The size is calculated as follows:
   4 + (8 + SubChunk1Size) + (8 + SubChunk2Size)
   This is the size of the rest of the chunk following this number.  This is the
   size of the entire file in bytes minus 8 bytes for the two fields not
   included in this count: ChunkID and ChunkSize.
Lastly, the size is followed by the format field, which should always be the
four ASCII characters "WAVE" for a well-formed WAV file.

Following the Riff header, the WAV file contains a "fmt" chunk and "data" chunk.
The "fmt" chunk describes the sound's data format and the "data" chunk contains
the actual data.
*/
type RiffHeader struct {
	*SubChunk
	Format uint32
}

/*
fmtChunk describes the format chunk of the sound file's data format. Like all
chunks, it starts with an ID and a size. (LE == Little Endian) It is followed
by:
  AudioFormat:   2 LE bytes. PCM = 1 (i.e. Linear quantization)
                 Values other than 1 indicate some form of compression.
  NumChannels:   2 LE bytes. Mono = 1, Stereo = 2, etc.
  SampleRate:    4 LE bytes. 8000, 44100, etc.
  ByteRate:      4 LE bytes. SampleRate * NumChannels * BitsPerSample/8
  BlockAlign:    2 LE bytes. NumChannels * BitsPerSample/8
                 The number of bytes for one sample including all channels.
  BitsPerSample: 2 LE bytes. 8 bits = 8, 16 bits = 16, etc.
*/
type fmtChunk struct {
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
}

/*
FmtChunk puts together the chunk ID and size provided by SubChunk with the
format data from fmtChunk. By defining them separately, the entire fmt chunk
can be filled in with a single call to binary.Read, since all pieces are little
endian. They are pulled back together in FmtChunk.
*/
type FmtChunk struct {
	*SubChunk
	*fmtChunk
}

/*
NewDefaultFmtChunk Returns a new FmtChunk with sensible defaults:
  2 channels
  16 bits per sample
  44100 sample rate
  176400 byte rate
  4 byte block allignment
*/
func NewDefaultFmtChunk() *FmtChunk {
	return &FmtChunk{
		&SubChunk{
			Id:   uint32(1718449184),
			Size: uint32(16),
		},
		&fmtChunk{
			AudioFormat:   uint16(1),
			NumChannels:   uint16(2),
			SampleRate:    uint32(44100),
			ByteRate:      uint32(176400),
			BlockAlign:    uint16(4),
			BitsPerSample: uint16(16),
		},
	}
}

/*
DataChunk contains the actual sound data. The ID of the data chunk should always
be the four ASCII characters "data".
*/
type DataChunk struct {
	*SubChunk
	Samples []Sample
}

/*
Sample is simply a slice of slices of bytes. Each sample is an entry in the
top-level array, while the data for each channel in each sample is stored as
bytes in the second-level array.
*/
type Sample [][]byte

// Wav stores a WAV file's format and data.
type Wav struct {
	Riff *RiffHeader
	Fmt  *FmtChunk
	Data *DataChunk
}

// WavReader contains the wav content as well as an internal buffer for reading
// contents from the file.
type WavReader struct {
	*Wav
	buffer io.Reader
}

// WavWriter contains the basic wav information as well as the buffer being
// written to.
type WavWriter struct {
	*Wav
	buffer io.WriterAt
}

/*
uint32AsString will return a string value, given a uint32, where each byte is
interpreted as an ASCII character. Bytes are interpreted in big endian format.
*/
func uint32AsString(number *uint32) string {
	var buffer bytes.Buffer
	binary.Write(&buffer, binary.BigEndian, number)
	return buffer.String()
}

/*
readSubChunk reads and returns a populated SubChunk given an *io.Reader to read
from. An error is returned from this function if there was an error in reading
the necessary bytes.
*/
func readSubChunk(reader *io.Reader) (*SubChunk, error) {
	newSubChunk := &SubChunk{}
	if err := binary.Read(
		*reader, binary.BigEndian, &newSubChunk.Id); err != nil {
		return nil, err
	}
	if err := binary.Read(
		*reader, binary.LittleEndian, &newSubChunk.Size); err != nil {
		return nil, err
	}
	return newSubChunk, nil
}

/*
readRiffHeader reads and returns a populated RiffHeader given an *io.Reader to
read from. The method will attempt to validate the header meets the standard WAV
file format. Returns a non-nil error when there is a validation error or a
problem reading the data.
*/
func readRiffHeader(reader *io.Reader) (*RiffHeader, error) {
	var err error
	var subChunk *SubChunk

	// Read the SubChunk of the Riff header.
	subChunk, err = readSubChunk(reader)
	if err != nil {
		return nil, err
	}
	// Validate the Riff header chunk ID.
	if uintString := uint32AsString(&subChunk.Id); uintString != Riff {
		return nil, fmt.Errorf(RiffError, uintString)
	}
	// Create a Riff header.
	riffHeader := &RiffHeader{SubChunk: subChunk}
	if err := binary.Read(
		*reader, binary.BigEndian, &riffHeader.Format); err != nil {
		return nil, fmt.Errorf("error reading format: %v", err.Error())
	}
	// Validate the format field of the Riff header.
	if uintString := uint32AsString(&riffHeader.Format); uintString != Wave {
		return nil, fmt.Errorf(WaveError, uintString)
	}
	return riffHeader, nil
}

/*
readFormatChunk reads and returns a populated FormatChunk given an *io.Reader to
read from. Returns a non-nil error when a problem is encountered reading the
data.
*/
func readFormatChunk(reader *io.Reader) (*FmtChunk, error) {
	var err error
	var subChunk *SubChunk

	// Read the SubChunk of the fmt chunk.
	subChunk, err = readSubChunk(reader)
	if err != nil {
		return nil, err
	}
	// Validate that the ID is "fmt ".
	if uintString := uint32AsString(&subChunk.Id); uintString != Fmt {
		return nil, fmt.Errorf(FmtError, uintString)
	}
	newFmtChunk := &fmtChunk{}
	if err := binary.Read(
		*reader, binary.LittleEndian, newFmtChunk); err != nil {
		return nil, err
	}
	return &FmtChunk{subChunk, newFmtChunk}, nil
}

/*
readDataChunk reads and returns a DataChunk. It also validates the "data" ID.
It does not completely read in all of the actual sound data immediately. Rather,
the sound data is returned by sequential calls to GetSample(). A non-nil error
is returned if there is a problem reading the data or the chunk is invalid.
*/
func readDataChunk(reader *io.Reader) (*DataChunk, error) {
	var err error
	var subChunk *SubChunk

	subChunk, err = readSubChunk(reader)
	if err != nil {
		return nil, err
	}
	// Validate that the ID is "data".
	if uintString := uint32AsString(&subChunk.Id); uintString != Data {
		return nil, fmt.Errorf(DataError, uintString)
	}
	return &DataChunk{SubChunk: subChunk}, nil
}

/**
NewWavReader reates a new, validated WavReader with initialized header data. If
the RIFF header does not indicate a WAV file, then this method will return a
non-nil error. Also, if the wav file's standard "fmt" block does not exist or
does not parse correctly, a non-nil error will be returned.
*/
func NewWavReader(r io.Reader) (*WavReader, error) {
	var riffHeader *RiffHeader
	var fmtChunk *FmtChunk
	var dataChunk *DataChunk
	var err error

	bufferedReader := io.Reader(bufio.NewReader(r))
	riffHeader, err = readRiffHeader(&bufferedReader)
	if err != nil {
		return nil, err
	}
	fmtChunk, err = readFormatChunk(&bufferedReader)
	if err != nil {
		return nil, err
	}
	dataChunk, err = readDataChunk(&bufferedReader)
	if err != nil {
		return nil, err
	}
	return &WavReader{
		&Wav{riffHeader, fmtChunk, dataChunk}, bufferedReader}, nil
}

/*
Gathers audio samples from a WAV file's data chunk. As samples are read, they
re appended to the WavReader's DataChunk. The number of slices of byte slices is
determined by the number of channels defined in the WAV file's fmt header. The
number of bytes in each slice is determined by the bits per sample defined in
the WAV file's fmt header.
*/
func (w *WavReader) GetSample() (Sample, error) {
	var channelSample []byte
	bytesPerSample := int(w.Fmt.BitsPerSample) / 8

	channels := make([][]byte, 0)
	for i := 0; i < int(w.Fmt.NumChannels); i++ {
		channelSample = make([]byte, bytesPerSample)
		for j := 0; j < bytesPerSample; j++ {
			if err := binary.Read(
				w.buffer,
				binary.LittleEndian,
				&channelSample[j]); err != nil {
				return nil, err
			}
		}
		channels = append(channels, channelSample)
	}
	newSample := Sample(channels)
	w.Data.Samples = append(w.Data.Samples, newSample)
	return Sample(channels), nil
}

/*
NewWavWriter Returns a WavWriter that can be used to create a wav file. It
requires a WriterAt so that information in the header can be updated as samples
are added to the WAV file. The passed in FormatChunk will define whether or not
samples passed to this writer are valid.
*/
func NewWavWriter(output io.WriterAt, fmt *FmtChunk) (*WavWriter, error) {
	if fmt == nil {
		fmt = NewDefaultFmtChunk()
	}
	wavWriter := &WavWriter{&Wav{
		defaultRiffHeader,
		fmt,
		defaultDataChunk,
	}, output}
	if err := wavWriter.writeInitialData(); err != nil {
		return nil, err
	}
	return wavWriter, nil
}

/*
writeInitialData writes the initial riff header, fmt chunk and data chunk to the
WavWriter.
*/
func (w *WavWriter) writeInitialData() error {
	var buffer = new(bytes.Buffer)
	var err error

	// Write the RIFF header.
	buffer.Reset()
	binary.Write(buffer, binary.BigEndian, w.Riff.Id)
	if _, err = w.buffer.WriteAt(buffer.Bytes(), int64(0)); err != nil {
		return err
	}
	buffer.Reset()
	binary.Write(buffer, binary.LittleEndian, w.Riff.Size)
	if _, err = w.buffer.WriteAt(buffer.Bytes(), int64(4)); err != nil {
		return err
	}
	buffer.Reset()
	binary.Write(buffer, binary.BigEndian, w.Riff.Format)
	if _, err = w.buffer.WriteAt(buffer.Bytes(), int64(8)); err != nil {
		return err
	}

	// Write the format chunk.
	buffer.Reset()
	binary.Write(buffer, binary.BigEndian, w.Fmt.Id)
	if _, err = w.buffer.WriteAt(buffer.Bytes(), int64(12)); err != nil {
		return err
	}
	buffer.Reset()
	binary.Write(buffer, binary.LittleEndian, w.Fmt.Size)
	if _, err = w.buffer.WriteAt(buffer.Bytes(), int64(16)); err != nil {
		return err
	}
	buffer.Reset()
	binary.Write(buffer, binary.LittleEndian, w.Fmt.fmtChunk)
	if _, err = w.buffer.WriteAt(buffer.Bytes(), int64(20)); err != nil {
		return err
	}

	// Write the data chunk.
	buffer.Reset()
	binary.Write(buffer, binary.BigEndian, w.Data.Id)
	if _, err = w.buffer.WriteAt(buffer.Bytes(), int64(36)); err != nil {
		return err
	}
	buffer.Reset()
	binary.Write(buffer, binary.LittleEndian, w.Data.Size)
	if _, err = w.buffer.WriteAt(buffer.Bytes(), int64(40)); err != nil {
		return err
	}
	return nil
}

/*
Adds a sample to the WavWriter's data chunk. The sample is validated against the
information in the fmt header. A non-nil error is returned if there is a problem
writing the Sample or if the sample is invalid.
*/
func (w *WavWriter) AddSample(sample Sample) error {
	if samples := len(sample); samples != int(w.Fmt.NumChannels) {
		return fmt.Errorf(ChannelError, w.Fmt.NumChannels, samples)
	}

	expectedBytes := (w.Fmt.BitsPerSample / 8) * w.Fmt.NumChannels
	var counted int
	for index := range sample {
		counted += len(sample[index])
	}
	if counted != int(expectedBytes) {
		return fmt.Errorf(SampleError, expectedBytes, counted)
	}

	// Write the new sizes and the new sample. The sample must be written
	// before the data size gets updated so the correct data offset can be
	// calculated.
	var buffer = new(bytes.Buffer)
	var err error

	buffer.Reset()
	for i := range sample {
		for j := range sample[i] {
			binary.Write(buffer, binary.LittleEndian, sample[i][j])
		}
	}
	offset := DataOffset + int64(w.Data.Size)
	_, err = w.buffer.WriteAt(buffer.Bytes(), int64(offset))
	if err != nil {
		return err
	}

	// Add the data to the WavWriter and update the counts.
	w.Data.Samples = append(w.Data.Samples, sample)
	w.Riff.Size += uint32(counted)
	w.Data.Size += uint32(counted)

	buffer.Reset()
	binary.Write(buffer, binary.LittleEndian, w.Riff.Size)
	_, err = w.buffer.WriteAt(buffer.Bytes(), int64(RiffSizeOffset))
	if err != nil {
		return err
	}
	buffer.Reset()
	binary.Write(buffer, binary.LittleEndian, w.Data.Size)
	_, err = w.buffer.WriteAt(buffer.Bytes(), int64(DataSizeOffset))
	if err != nil {
		return err
	}

	return nil
}
