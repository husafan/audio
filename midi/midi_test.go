package midi_test

import (
	"bytes"
	"regexp"
	"testing"

	. "github.com/husafan/audio/midi"
	"github.com/stretchr/testify/assert"
)

func TestVariableLengthQuantity(t *testing.T) {
	value := ReadVariableLengthQuantity(bytes.NewBuffer([]byte{0x7F}))
	assert.Equal(t, uint64(127), value)

	value = ReadVariableLengthQuantity(bytes.NewBuffer([]byte{0x81, 0x48}))
	assert.Equal(t, uint64(200), value)

	value = ReadVariableLengthQuantity(bytes.NewBuffer([]byte{0xFF, 0xFF, 0x7F}))
	assert.Equal(t, uint64(2097151), value)

	value = ReadVariableLengthQuantity(bytes.NewBuffer([]byte{0x81, 0x80, 0x80, 0x00}))
	assert.Equal(t, uint64(2097152), value)

	value = ReadVariableLengthQuantity(bytes.NewBuffer([]byte{0xC0, 0x80, 0x80, 0x00}))
	assert.Equal(t, uint64(134217728), value)
}

func TestMidiHeaderIncorrectSize(t *testing.T) {
	var buffer bytes.Buffer
	buffer.WriteString("MThd")
	// Write the header length. Should always be 16, but is 5 here.
	buffer.Write([]byte{0, 0, 0, 5})

	midi := new(Midi)
	err := midi.UnmarshalBinary(buffer.Bytes())
	assert.NotNil(t, err)
	re := regexp.MustCompile("expected a header length of 16 but found a length of 5")
	assert.NotEqual(t, "", re.FindString(err.Error()))
}

func TestMidiHeaderChunkTooSmall(t *testing.T) {
	var buffer bytes.Buffer
	buffer.WriteString("MTh")

	midi := new(Midi)
	err := midi.UnmarshalBinary(buffer.Bytes())
	assert.NotNil(t, err)
	re := regexp.MustCompile("EOF")
	assert.NotEqual(t, "", re.FindString(err.Error()))

	buffer.Reset()
	buffer.WriteString("MThd")
	buffer.Write([]byte{0, 0, 0, 16})

	midi = new(Midi)
	err = midi.UnmarshalBinary(buffer.Bytes())
	assert.NotNil(t, err)
	assert.NotEqual(t, "", re.FindString(err.Error()))
}

func TestMidiHeaderChunkParsed(t *testing.T) {

}
