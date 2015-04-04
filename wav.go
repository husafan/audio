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
type subChunk struct {
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
	*subChunk
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
// file is not a valid WAV file, then this method will return a non-nil error.
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

func (w *WavReader) readFormatChunk() (*FormatChunk, error) {
	newSubChunk, err := w.ReadSubChunk()
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

func (w *WavReader) ReadSubChunk() (*subChunk, error) {
	newSubChunk := &subChunk{}
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
