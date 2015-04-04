package wav_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"regexp"
	"strings"
	"testing"

	. "github.com/husafan/wav"
	"github.com/stretchr/testify/assert"
)

func TestErrorReadingFourCC(t *testing.T) {
	// FourCC (four character code) must be 4 bytes.
	data := strings.NewReader("RIF")
	reader, err := NewWavReader(data)
	assert.Nil(t, reader)
	assert.NotNil(t, err)
	re := regexp.MustCompile("FourCC")
	assert.NotNil(t, re.FindString(err.Error()))
}

func TestErrorReadingSize(t *testing.T) {
	// Chunk ID must be 4 bytes.
	data := strings.NewReader("RIFFd")
	reader, err := NewWavReader(data)
	assert.Nil(t, reader)
	assert.NotNil(t, err)
	re := regexp.MustCompile("Size")
	assert.NotNil(t, re.FindString(err.Error()))
}

func TestErrorReadingFormat(t *testing.T) {
	// Chunk ID must be 4 bytes.
	data := strings.NewReader("RIFFaaaaWAV")
	reader, err := NewWavReader(data)
	assert.Nil(t, reader)
	assert.NotNil(t, err)
	re := regexp.MustCompile("Format")
	assert.NotNil(t, re.FindString(err.Error()))
}

func TestInvalidWavFileChunkType(t *testing.T) {
	data := strings.NewReader("Hello World!")
	reader, err := NewWavReader(data)
	assert.Nil(t, reader)
	assert.NotNil(t, err)
	assert.Equal(t, fmt.Sprintf(RiffError, "Hell"), err.Error())
}

func TestInvalidWavFileFormatType(t *testing.T) {
	data := strings.NewReader("RIFFo World!")
	reader, err := NewWavReader(data)
	assert.Nil(t, reader)
	assert.NotNil(t, err)
	assert.Equal(t, fmt.Sprintf(WaveError, "rld!"), err.Error())
}

func TestReadHeaderChunk(t *testing.T) {
	var buffer bytes.Buffer
	// Build a fake RIFF chunk
	buffer.WriteString("RIFF")
	var sizeBytes = make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, uint32(12345))
	buffer.Write(sizeBytes)
	buffer.WriteString("WAVE")

	_, err := NewWavReader(strings.NewReader(buffer.String()))
	// Expect an invalid Format Chunk as the header was parsed successfully.
	re := regexp.MustCompile("Invalid format chunk")
	assert.NotNil(t, err)
	assert.NotNil(t, re.FindString(err.Error()))
}

func TestInvalidFormatChunk(t *testing.T) {
	var buffer bytes.Buffer
	var sizeBytes = make([]byte, 4)
	wavSize := uint32(123)
	fmtChunkId := uint32(456)

	// Build a fake RIFF chunk
	buffer.WriteString("RIFF")
	binary.LittleEndian.PutUint32(sizeBytes, wavSize)
	buffer.Write(sizeBytes)
	buffer.WriteString("WAVE")

	// Build a malformed Format chunk with too little data.
	binary.BigEndian.PutUint32(sizeBytes, fmtChunkId)
	buffer.Write(sizeBytes)

	data := strings.NewReader(buffer.String())
	reader, err := NewWavReader(data)
	assert.Nil(t, reader)
	assert.NotNil(t, err)
}

func TestValidHeaderAndFormatChunk(t *testing.T) {
	var buffer bytes.Buffer
	var size4Bytes = make([]byte, 4)
	var size2Bytes = make([]byte, 2)
	wavSize := uint32(123)
	fmtChunkId := uint32(456)
	fmtChunkSize := uint32(789)
	audioFormat := uint16(111)
	numChannels := uint16(5)
	sampleRate := uint32(44000)
	byteRate := uint32(56000)
	blockAlign := uint16(12)
	bitsPerSample := uint16(2200)

	buffer.WriteString("RIFF")
	binary.LittleEndian.PutUint32(size4Bytes, wavSize)
	buffer.Write(size4Bytes)
	buffer.WriteString("WAVE")
	binary.BigEndian.PutUint32(size4Bytes, fmtChunkId)
	buffer.Write(size4Bytes)
	binary.LittleEndian.PutUint32(size4Bytes, fmtChunkSize)
	buffer.Write(size4Bytes)
	binary.LittleEndian.PutUint16(size2Bytes, audioFormat)
	buffer.Write(size2Bytes)
	binary.LittleEndian.PutUint16(size2Bytes, numChannels)
	buffer.Write(size2Bytes)
	binary.LittleEndian.PutUint32(size4Bytes, sampleRate)
	buffer.Write(size4Bytes)
	binary.LittleEndian.PutUint32(size4Bytes, byteRate)
	buffer.Write(size4Bytes)
	binary.LittleEndian.PutUint16(size2Bytes, blockAlign)
	buffer.Write(size2Bytes)
	binary.LittleEndian.PutUint16(size2Bytes, bitsPerSample)
	buffer.Write(size2Bytes)

	data := strings.NewReader(buffer.String())
	reader, err := NewWavReader(data)
	assert.Nil(t, err)
	assert.NotNil(t, reader)
	assert.Equal(t, wavSize, reader.Size)
	assert.Equal(t, fmtChunkId, reader.FormatChunk.Id)
	assert.Equal(t, fmtChunkSize, reader.FormatChunk.Size)
	assert.Equal(t, audioFormat, reader.FormatChunk.AudioFormat)
	assert.Equal(t, numChannels, reader.FormatChunk.NumChannels)
	assert.Equal(t, sampleRate, reader.FormatChunk.SampleRate)
	assert.Equal(t, byteRate, reader.FormatChunk.ByteRate)
	assert.Equal(t, blockAlign, reader.FormatChunk.BlockAlign)
	assert.Equal(t, bitsPerSample, reader.FormatChunk.BitsPerSample)
}
