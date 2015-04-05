package wav

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	Riff             = "RIFF"
	RiffError        = "Invalid initial chunk ID of %s. Should be RIFF."
	Wave             = "WAVE"
	WaveError        = "Invalid format of %s. Should be WAVE."
	FormatChunkError = "Invalid format chunk: %s."
)

// An internal type for capturing the id and size of a subchunk.
type SubChunk struct {
	Id   uint32
	Size uint32
}

// An internal type for capturing the audio format described in the "fmt" chunk.
type formatChunk struct {
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
}

// A public type describing the "fmt" chunk. Describes the audio file format.
type FormatChunk struct {
	*SubChunk
	*formatChunk
}

// The base WavReader type. Stores information about the WAV's format as well
// as an internal buffer for reading chunks.
type WavReader struct {
	FormatChunk *FormatChunk
	FourCC      string
	Size        uint32
	Format      string
	buffer      io.Reader
}

// Creates a new, validated WavReader with initialized header data. If the RIFF
// header does not indicate a WAV file, then this method will return a non-nil
// error. Also, if the wav file's standard "fmt" block does not exist or does
// not parse correctly, a non-nil error will be returned.
func NewWavReader(r io.Reader) (*WavReader, error) {
	bufferedReader := bufio.NewReader(r)

	chunkId := [4]byte{}
	if err := binary.Read(
		bufferedReader, binary.BigEndian, &chunkId); err != nil {
		return nil, errors.New(
			fmt.Sprintf("Error reading FourCC: %v", err.Error()))
	}

	chunkIdStr := string(chunkId[:])
	// The first chunk ID for a valid WAV file should be "RIFF".
	if chunkIdStr != Riff {
		return nil, errors.New(fmt.Sprintf(RiffError, chunkIdStr))
	}

	var size uint32
	if err := binary.Read(
		bufferedReader, binary.LittleEndian, &size); err != nil {
		return nil, errors.New(
			fmt.Sprintf("Error reading Size: %v", err.Error()))
	}

	format := [4]byte{}
	if err := binary.Read(
		bufferedReader, binary.BigEndian, &format); err != nil {
		return nil, errors.New(
			fmt.Sprintf("Error reading Format: %v", err.Error()))
	}
	formatStr := string(format[:])
	// The format for the RIFF chunk of a valid wav file should be WAVE.
	if formatStr != Wave {
		return nil, errors.New(fmt.Sprintf(WaveError, formatStr))
	}

	wavReader := &WavReader{
		FourCC: chunkIdStr,
		Size:   size,
		Format: formatStr,
		// The buffer is kept internal.
		buffer: bufferedReader,
	}
	formatChunk, err := wavReader.readFormatChunk()
	if err != nil {
		return nil, errors.New(fmt.Sprintf(FormatChunkError, err.Error()))
	}
	wavReader.FormatChunk = formatChunk
	return wavReader, nil
}

/**
 * Gathers audio samples from a WAV file's data chunk. Returns an array with
 * [w.NumChannels] entries with each entry containing [w.BitsPerSample] data.
 * In most cases, each entry corresponds to 16 bits per entry.
 */
func (w *WavReader) GetSample() ([]int16, error) {
	numChannels := int(w.FormatChunk.NumChannels)
	sample := make([]int16, numChannels)
	for i := 0; i < numChannels; i++ {
		if err := binary.Read(
			w.buffer, binary.LittleEndian, &sample[i]); err != nil {
			return nil, err
		}
	}
	return sample, nil
}

func (w *WavReader) readFormatChunk() (*FormatChunk, error) {
	newSubChunk, err := w.readSubChunk()
	if err != nil {
		return nil, err
	}
	newFormatChunk := &formatChunk{}
	if err := binary.Read(
		w.buffer, binary.LittleEndian, newFormatChunk); err != nil {
		return nil, err
	}
	return &FormatChunk{newSubChunk, newFormatChunk}, nil
}

func (w *WavReader) readSubChunk() (*SubChunk, error) {
	newSubChunk := &SubChunk{}
	if err := binary.Read(
		w.buffer, binary.BigEndian, &newSubChunk.Id); err != nil {
		return nil, err
	}
	if err := binary.Read(
		w.buffer, binary.LittleEndian, &newSubChunk.Size); err != nil {
		return nil, err
	}
	return newSubChunk, nil
}
