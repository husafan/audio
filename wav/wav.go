package wav

/**
 * WAV File Format implemented within: http://goo.gl/Wi3NNU.
 */

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	Data             string = "data"
	DataError        string = "Invalid data chunk ID of %s. Should be data."
	Fmt              string = "fmt "
	FmtError         string = "Invalid fmt chunk ID of %s. Should be 'fmt '."
	Riff             string = "RIFF"
	RiffError        string = "Invalid initial chunk ID of %s. Should be RIFF."
	SampleError      string = "Expected %v bytes per sample. Only found %v."
	Wave             string = "WAVE"
	WaveError        string = "Invalid format of %s. Should be WAVE."
	FormatChunkError string = "Invalid format chunk: %s."
	RiffSizeOffset   int64  = 4
	DataSizeOffset   int64  = 40
	DataOffset       int64  = 44
)

/**
 * A default Riff header. It conains an ID of 'RIFF' and a format of 'WAVE'. It
 * contains a default size of 36. Every time a sample is added, the size should
 * be incremented by the number BytesPerSample * NumOfChannels.
 */
var defaultRiffHeader *RiffHeader = &RiffHeader{
	&SubChunk{
		Id:   uint32(1380533830),
		Size: uint32(36),
	},
	uint32(1463899717),
}

/**
 * A default data chunk. It has an ID of 'data' and a size of 0 with no
 * samples to start with.
 */
var defaultDataChunk *DataChunk = &DataChunk{
	&SubChunk{
		Id:   uint32(1684108385),
		Size: uint32(0),
	},
	make([]Sample, 0),
}

/**
 * All chunks within a WAV file contain an ID and a size in bytes. The ID
 * is big-endian while the size is little-endian.
 */
type SubChunk struct {
	Id   uint32
	Size uint32
}

/**
 * The Riff header chunk is the first chunk in a well-formed WAV file. The ID
 * of the chunk is always the four ASCII characters "RIFF". It is followed by
 * the size of the entire WAV file. The size is calculated as follows:
 *     4 + (8 + SubChunk1Size) + (8 + SubChunk2Size)
 *     This is the size of the rest of the chunk following this number.  This
 *     is the size of the entire file in bytes minus 8 bytes for the two fields
 *     not included in this count: ChunkID and ChunkSize.
 * Lastly, the size is followed by the format field, which should always be the
 * four ASCII characters "WAVE" for a well-formed WAV file.
 *
 * Following the Riff header, the WAV file contains a "fmt" chunk and "data"
 * chunk. The "fmt" chunk describes the sound's data format and the "data" chunk
 * contains the actual data.
 */
type RiffHeader struct {
	*SubChunk
	Format uint32
}

/**
 * The "fmt" chunk describes the sound's data format. Like all chunks, it starts
 * with an ID and a size. (LE == Little Endian) It is followed by:
 *   AudioFormat:   2 LE bytes. PCM = 1 (i.e. Linear quantization)
 *                  Values other than 1 indicate some form of compression.
 *   NumChannels:   2 LE bytes. Mono = 1, Stereo = 2, etc.
 *   SampleRate:    4 LE bytes. 8000, 44100, etc.
 *   ByteRate:      4 LE bytes. SampleRate * NumChannels * BitsPerSample/8
 *   BlockAlign:    2 LE bytes. NumChannels * BitsPerSample/8
 *                  The number of bytes for one sample including all channels.
 *   BitsPerSample: 2 LE bytes. 8 bits = 8, 16 bits = 16, etc.
 */
type fmtChunk struct {
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
}

/**
 * The FmtChunk puts together the chunk ID and size provided by SubChunk with
 * the format data from fmtChunk. By defining them separately, the entire fmt
 * chunk can be filled in with a single call to binary.Read, since all pieces
 * are little endian. They are pulled back together in FmtChunk.
 */
type FmtChunk struct {
	*SubChunk
	*fmtChunk
}

/**
 * Returns a new FmtChunk with sensible defaults.
 * @return {*FmtChunk} A FmtChunk pointer with the following defaults:
 *     2 channels
 *     16 bits per sample
 *     44100 sample rate
 *     176400 byte rate
 *     4 byte block allignment
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

/**
 * The DataChunk contains the actual sound data. The ID of the data chunk
 * should always be the four ASCII characters "data".
 */
type DataChunk struct {
	*SubChunk
	Samples []Sample
}

/**
 * The Sample type is simply an array of byte arrays. Each sample is an entry
 * in the top-level array, while the data for each channel in each sample is
 * stored as bytes in the second-level array.
 */
type Sample [][]byte

// The base wav type stores information about the WAV's format.
type Wav struct {
	Riff *RiffHeader
	Fmt  *FmtChunk
	Data *DataChunk
}

// The WavReader contains the wav content as well as an internal buffer for
// reading contents from the file.
type WavReader struct {
	*Wav
	buffer io.Reader
}

// The WavWriter contains the basic wav information as well as the buffer being
// written to.
type WavWriter struct {
	*Wav
	buffer io.WriterAt
}

/**
 * Given a uint32, will return the string value where each byte is interpreted
 * as an ASCII character. Bytes are interpreted in big endian format.
 * @param {*uint32} The number to interpret.
 */
func uint32AsString(number *uint32) string {
	buffer := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buffer, binary.BigEndian, number)
	return buffer.String()
}

/**
 * A utility method for reading a SubChunk. An error is returned from this
 * function if there was an error in reading the necessary bytes.
 * @param {*io.Reader} A reader encapsulating the data to be read.
 * @return {*SubChunk, error} Returns a pointer to a SubChunk and a nil error
 *      when successful, or a nil SubChunk and an error on failure.
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

/**
 * A utility method for reading and validating the Riff header of a wav file.
 * @param {*io.Reader} A reader encapsulating the data to be read.
 * @return {*SubChunk, error} Returns a pointer to a RiffHeader and a nil error
 *      when successful, or a nil RiffHeader and an error on failure.
 */
func readRiffHeader(reader *io.Reader) (*RiffHeader, error) {
	var err error
	var subChunk *SubChunk
	var uintString string

	// Read the SubChunk of the Riff header.
	subChunk, err = readSubChunk(reader)
	if err != nil {
		return nil, err
	}
	// Validate the Riff header chunk ID.
	uintString = uint32AsString(&subChunk.Id)
	if uintString != Riff {
		return nil, errors.New(fmt.Sprintf(RiffError, uintString))
	}
	// Create a Riff header.
	riffHeader := &RiffHeader{subChunk, uint32(0)}
	if err := binary.Read(
		*reader, binary.BigEndian, &riffHeader.Format); err != nil {
		return nil, errors.New(
			fmt.Sprintf("Error reading Format: %v", err.Error()))
	}
	uintString = uint32AsString(&riffHeader.Format)
	// Validate the format field of the Riff header.
	if uintString != Wave {
		return nil, errors.New(fmt.Sprintf(WaveError, uintString))
	}
	return riffHeader, nil
}

/**
 * A function that reads and returns the "fmt" chunk of a wav file.
 * @param {*io.Reader} A reader encapsulating the data to be read.
 * @return {*SubChunk, error} Returns a pointer to a FmtChunk and a nil error
 *      when successful, or a nil FmtChunk and an error on failure.
 */
func readFormatChunk(reader *io.Reader) (*FmtChunk, error) {
	var err error
	var subChunk *SubChunk
	var uintString string

	// Read the SubChunk of the fmt chunk.
	subChunk, err = readSubChunk(reader)
	if err != nil {
		return nil, err
	}
	// Validate that the ID is "fmt ".
	uintString = uint32AsString(&subChunk.Id)
	if uintString != Fmt {
		return nil, errors.New(fmt.Sprintf(FmtError, uintString))
	}
	newFmtChunk := &fmtChunk{}
	if err := binary.Read(
		*reader, binary.LittleEndian, newFmtChunk); err != nil {
		return nil, err
	}
	return &FmtChunk{subChunk, newFmtChunk}, nil
}

/**
 * A function that reads, validates and returns the "data" chunk of a wav file.
 * @param {*io.Reader} A reader encapsulating the data to be read.
 * @return {*SubChunk, error} Returns a pointer to a DataChunk and a nil error
 *      when successful, or a nil DataChunk and an error on failure.
 */
func readDataChunk(reader *io.Reader) (*DataChunk, error) {
	var err error
	var subChunk *SubChunk
	var uintString string

	subChunk, err = readSubChunk(reader)
	if err != nil {
		return nil, err
	}
	// Validate that the ID is "data".
	uintString = uint32AsString(&subChunk.Id)
	if uintString != Data {
		return nil, errors.New(fmt.Sprintf(DataError, uintString))
	}
	return &DataChunk{subChunk, make([]Sample, 0)}, nil
}

/**
 * Creates a new, validated WavReader with initialized header data. If the RIFF
 * header does not indicate a WAV file, then this method will return a non-nil
 * error. Also, if the wav file's standard "fmt" block does not exist or does
 * not parse correctly, a non-nil error will be returned.
 * @param {io.Reader} A reader containing the WAV data.
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

/**
 * Gathers audio samples from a WAV file's data chunk. As samples are read, they
 * are also appended to the WavReader's DataChunk.
 * @return {*Sample, error} Returns a Sample per call and a nil error on
 *     succcess, or a nil Sample and an error on failure.
 */
func (w *WavReader) GetSample() (Sample, error) {
	var channelSample []byte
	bytesPerSample := int(w.Fmt.BitsPerSample) / 8

	channels := make([][]byte, 0)
	for i := 0; i < int(w.Fmt.NumChannels); i++ {
		channelSample = make([]byte, bytesPerSample)
		for j := 0; j < bytesPerSample; j++ {
			err := binary.Read(
				w.buffer,
				binary.LittleEndian,
				&channelSample[j])
			if err != nil {
				return nil, err
			}
		}
		channels = append(channels, channelSample)
	}
	newSample := Sample(channels)
	w.Data.Samples = append(w.Data.Samples, newSample)
	return Sample(channels), nil
}

/**
 * Returns a WavWriter that can be used to create a wav file.
 * @param {io.WriterAt} An instance of io.WriterAt that enables random
 *     access writing.
 * @param {FormatChunk} A format chunk describing the wav file. If none is given
 *     a default format chunk will be used.
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
	err := wavWriter.writeInitialData()
	if err != nil {
		return nil, err
	}
	return wavWriter, nil
}

/**
 * A private method on the WavWriter that writes the initial riff header, fmt
 * chunk and data chunk to the WriterAt.
 * @return {error} Returns an error if one was encountered during writing.
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

/**
 * Adds a sample to the WavWriter's data.
 * @return {error} Returns an error on failure, nil otherwise.
 */
func (w *WavWriter) AddSample(sample Sample) error {
	expectedBytes := (w.Fmt.BitsPerSample / 8) * w.Fmt.NumChannels
	var counted int
	for index := range sample {
		counted += len(sample[index])
	}
	if counted != int(expectedBytes) {
		return errors.New(
			fmt.Sprintf(SampleError, expectedBytes, counted))
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
