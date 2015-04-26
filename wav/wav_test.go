package wav_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"

	. "github.com/husafan/audio/wav"
	"github.com/stretchr/testify/assert"
)

var (
	wavSize       = uint32(123)
	fmtChunkId    = "fmt "
	fmtChunkSize  = uint32(789)
	audioFormat   = uint16(111)
	numChannels   = uint16(2)
	sampleRate    = uint32(44000)
	byteRate      = uint32(56000)
	blockAlign    = uint16(12)
	bitsPerSample = uint16(16)
)

func getValidHeaderAndFmtChunk() *bytes.Buffer {
	var buffer bytes.Buffer
	var size4Bytes = make([]byte, 4)
	var size2Bytes = make([]byte, 2)

	buffer.WriteString("RIFF")
	binary.LittleEndian.PutUint32(size4Bytes, wavSize)
	buffer.Write(size4Bytes)
	buffer.WriteString("WAVE")
	buffer.WriteString(fmtChunkId)
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

	return &buffer
}

func TestErrorReadingRiffIdNotEnoughtData(t *testing.T) {
	// FourCC (four character code) must be 4 bytes.
	data := strings.NewReader("RIF")
	reader, err := NewWavReader(data)
	assert.Nil(t, reader)
	assert.NotNil(t, err)
	re := regexp.MustCompile("EOF")
	assert.NotEqual(t, "", re.FindString(err.Error()))
}

func TestErrorReadingRiffWrongId(t *testing.T) {
	// FourCC (four character code) must be 4 bytes.
	data := strings.NewReader("RIFD0000")
	reader, err := NewWavReader(data)
	assert.Nil(t, reader)
	assert.NotNil(t, err)
	re := regexp.MustCompile("should be 'RIFF'")
	assert.NotEqual(t, "", re.FindString(err.Error()))
}

func TestErrorReadingSizeNotEnoughData(t *testing.T) {
	// Chunk ID must be 4 bytes.
	data := strings.NewReader("RIFFd")
	reader, err := NewWavReader(data)
	assert.Nil(t, reader)
	assert.NotNil(t, err)
	re := regexp.MustCompile("EOF")
	assert.NotEqual(t, "", re.FindString(err.Error()))
}

func TestErrorReadingFormatNotEnoughData(t *testing.T) {
	// Chunk ID must be 4 bytes.
	data := strings.NewReader("RIFFaaaaWAV")
	reader, err := NewWavReader(data)
	assert.Nil(t, reader)
	assert.NotNil(t, err)
	re := regexp.MustCompile("EOF")
	assert.NotEqual(t, "", re.FindString(err.Error()))
}

func TestErrorReadingFormatWrongValue(t *testing.T) {
	// Chunk ID must be 4 bytes.
	data := strings.NewReader("RIFFaaaaWAVD")
	reader, err := NewWavReader(data)
	assert.Nil(t, reader)
	assert.NotNil(t, err)
	re := regexp.MustCompile("should be 'WAVE'")
	assert.NotEqual(t, "", re.FindString(err.Error()))
}

func TestInvalidFormatChunkNotEnoughData(t *testing.T) {
	var buffer bytes.Buffer
	// Build a fake RIFF chunk
	buffer.WriteString("RIFF")
	var sizeBytes = make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, uint32(12345))
	buffer.Write(sizeBytes)
	buffer.WriteString("WAVE")

	_, err := NewWavReader(strings.NewReader(buffer.String()))
	// Expect an invalid Format Chunk as the header was parsed successfully.
	assert.NotNil(t, err)
	re := regexp.MustCompile("EOF")
	assert.NotEqual(t, "", re.FindString(err.Error()))
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

	reader, err := NewWavReader(&buffer)
	assert.Nil(t, reader)
	assert.NotNil(t, err)
}

func TestInvalidDataChunkNotEnoughData(t *testing.T) {
	buffer := getValidHeaderAndFmtChunk()

	reader, err := NewWavReader(buffer)
	assert.Nil(t, reader)
	assert.NotNil(t, err)
	re := regexp.MustCompile("EOF")
	assert.NotEqual(t, "", re.FindString(err.Error()))
}

func TestValidHeaderFormatAndDataChunk(t *testing.T) {
	var size4Bytes = make([]byte, 4)

	buffer := getValidHeaderAndFmtChunk()
	buffer.WriteString("data")
	binary.LittleEndian.PutUint32(size4Bytes, uint32(0))
	buffer.Write(size4Bytes)

	reader, err := NewWavReader(buffer)
	assert.Nil(t, err)
	assert.NotNil(t, reader)
	assert.Equal(t, wavSize, reader.Riff.Size)

	var fmtChunkStr = make([]byte, 4)
	binary.Write(
		bytes.NewBuffer(fmtChunkStr), binary.BigEndian, reader.Fmt.Id)
	assert.Equal(t, fmtChunkId, string(fmtChunkId))
	assert.Equal(t, fmtChunkSize, reader.Fmt.Size)
	assert.Equal(t, audioFormat, reader.Fmt.AudioFormat)
	assert.Equal(t, numChannels, reader.Fmt.NumChannels)
	assert.Equal(t, sampleRate, reader.Fmt.SampleRate)
	assert.Equal(t, byteRate, reader.Fmt.ByteRate)
	assert.Equal(t, blockAlign, reader.Fmt.BlockAlign)
	assert.Equal(t, bitsPerSample, reader.Fmt.BitsPerSample)
}

func TestValidData(t *testing.T) {
	var size2Bytes = make([]byte, 2)
	var size4Bytes = make([]byte, 4)

	buffer := getValidHeaderAndFmtChunk()
	buffer.WriteString("data")
	binary.LittleEndian.PutUint32(size4Bytes, uint32(0))
	buffer.Write(size4Bytes)

	// Write data entries corresponding to 2 channels and 16 bits per
	// sample.
	binary.LittleEndian.PutUint16(size2Bytes, 123)
	buffer.Write(size2Bytes)
	var signedSample int16 = -123
	binary.LittleEndian.PutUint16(size2Bytes, uint16(signedSample))
	buffer.Write(size2Bytes)
	binary.LittleEndian.PutUint16(size2Bytes, 321)
	buffer.Write(size2Bytes)
	signedSample = -321
	binary.LittleEndian.PutUint16(size2Bytes, uint16(signedSample))
	buffer.Write(size2Bytes)

	reader, err := NewWavReader(buffer)
	assert.Nil(t, err)
	assert.NotNil(t, reader)

	var value uint16
	sample, err := reader.GetSample()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(sample))
	// sample[0] and sample[1] should each have 2 bytes.
	assert.Equal(t, 2, len(sample[0]))
	assert.Equal(t, 2, len(sample[1]))
	// Confirm the actual sample values per channel.
	binary.Read(bytes.NewBuffer(sample[0]), binary.LittleEndian, &value)
	assert.Equal(t, int16(123), int16(value))
	binary.Read(bytes.NewBuffer(sample[1]), binary.LittleEndian, &value)
	assert.Equal(t, int16(-123), int16(value))

	sample, err = reader.GetSample()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(sample))
	// sample[0] and sample[1] should each have 2 bytes.
	assert.Equal(t, 2, len(sample[0]))
	assert.Equal(t, 2, len(sample[1]))

	binary.Read(bytes.NewBuffer(sample[0]), binary.LittleEndian, &value)
	assert.Equal(t, int16(321), int16(value))
	binary.Read(bytes.NewBuffer(sample[1]), binary.LittleEndian, &value)
	assert.Equal(t, int16(-321), int16(value))

	sample, err = reader.GetSample()
	assert.Nil(t, sample)
	assert.NotNil(t, err)
	assert.Equal(t, io.EOF, err)

	// Confirm that the Samples have been added to the WavReader.
	assert.Equal(t, 2, len(reader.Data.Samples))
}

type mockWriterAtCloser struct {
	data []byte
}

func (m *mockWriterAtCloser) WriteAt(p []byte, off int64) (n int, err error) {
	if int(off)+len(p) > len(m.data) {
		return 0, errors.New("Buffer not big enough.")
	}
	for index, value := range p {
		m.data[off+int64(index)] = value
	}
	return len(p), nil
}

func TestWavWriterErrorNotEnoughBuffer(t *testing.T) {
	writer := &mockWriterAtCloser{make([]byte, 10)}
	wavWriter, err := NewWavWriter(writer, nil)
	assert.Nil(t, wavWriter)
	assert.NotNil(t, err)
}

func TestWavWriterValidDefaultHeader(t *testing.T) {
	writer := &mockWriterAtCloser{make([]byte, 100)}
	wavWriter, err := NewWavWriter(writer, nil)
	assert.NotNil(t, wavWriter)
	assert.Nil(t, err)

	var num16 uint16
	var num32 uint32
	assert.Equal(t, "RIFF", string(writer.data[:4]))
	binary.Read(
		bytes.NewBuffer(writer.data[4:8]), binary.LittleEndian, &num32)
	assert.Equal(t, uint32(36), num32)
	assert.Equal(t, "WAVE", string(writer.data[8:12]))
	assert.Equal(t, "fmt ", string(writer.data[12:16]))
	binary.Read(
		bytes.NewBuffer(writer.data[16:20]), binary.LittleEndian,
		&num32)
	assert.Equal(t, uint32(16), num32)
	binary.Read(
		bytes.NewBuffer(writer.data[20:22]), binary.LittleEndian,
		&num16)
	assert.Equal(t, uint16(1), num16)
	binary.Read(
		bytes.NewBuffer(writer.data[22:24]), binary.LittleEndian,
		&num16)
	assert.Equal(t, uint16(2), num16)
	binary.Read(
		bytes.NewBuffer(writer.data[24:28]), binary.LittleEndian,
		&num32)
	assert.Equal(t, uint32(44100), num32)
	binary.Read(
		bytes.NewBuffer(writer.data[28:32]), binary.LittleEndian,
		&num32)
	assert.Equal(t, uint32(176400), num32)
	binary.Read(
		bytes.NewBuffer(writer.data[32:34]), binary.LittleEndian,
		&num16)
	assert.Equal(t, uint16(4), num16)
	binary.Read(
		bytes.NewBuffer(writer.data[34:36]), binary.LittleEndian,
		&num16)
	assert.Equal(t, uint16(16), num16)
	assert.Equal(t, "data", string(writer.data[36:40]))
	binary.Read(
		bytes.NewBuffer(writer.data[40:44]), binary.LittleEndian,
		&num32)
	assert.Equal(t, uint32(0), num32)
}

func TestWrongChannelSampleSize(t *testing.T) {
	writer := &mockWriterAtCloser{make([]byte, 100)}
	wavWriter, err := NewWavWriter(writer, nil)
	wavWriter.Fmt.NumChannels = uint16(1)
	wavWriter.Fmt.BitsPerSample = uint16(8)

	sample := Sample([][]byte{{1}, {2}})
	err = wavWriter.AddSample(sample)
	assert.NotNil(t, err)
	re := regexp.MustCompile("expected 1 channels; found 2")
	assert.NotEqual(t, "", re.FindString(err.Error()))
}

func TestWrongSampleSize(t *testing.T) {
	writer := &mockWriterAtCloser{make([]byte, 100)}
	wavWriter, err := NewWavWriter(writer, nil)

	// Default writer expects 4 bytes per sample.
	sample := Sample([][]byte{{1}, {2}})
	err = wavWriter.AddSample(sample)
	assert.NotNil(t, err)
	re := regexp.MustCompile("per sample but only found 2")
	assert.NotEqual(t, "", re.FindString(err.Error()))
}

func TestAddSamples(t *testing.T) {
	writer := &mockWriterAtCloser{make([]byte, 100)}
	wavWriter, err := NewWavWriter(writer, nil)

	// Default writer expects 4 bytes per sample.
	sample := Sample([][]byte{{1, 2}, {2, 3}})
	err = wavWriter.AddSample(sample)
	assert.Nil(t, err)

	// Check the new sizes.
	var num32 uint32
	binary.Read(
		bytes.NewBuffer(writer.data[4:8]), binary.LittleEndian, &num32)
	assert.Equal(t, uint32(40), num32)
	binary.Read(
		bytes.NewBuffer(writer.data[40:44]), binary.LittleEndian,
		&num32)
	assert.Equal(t, uint32(4), num32)
	// Confirm the sample was written.
	assert.Equal(t, writer.data[44:48], []byte{1, 2, 2, 3})
}
