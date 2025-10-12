package device

// The number of items to buffer in each stream
const DEVICE_STREAM_BUFFER_SIZE = 128

// Interface for audio input devices, e.g. microphones
//
// Input devices need only define some way to get data out of the device,
// which returns a channel (stream) of PCMFrames
type AudioInputDevice interface {
	// Get the input stream of this audio device.
	//
	// Raw audio data (as PCMFrames) will arrive on the returned channel.
	GetStream() <-chan []int16
	NumChannels() int
	SampleRate() int
}

// Interface for audio output devices, e.g. speakers
//
// Output devices need only define some way to consume data,
// taken as a channel (stream) of PCMFrames
type AudioOutputDevice interface {
	// Set the data stream of this audio device.
	//
	// Raw audio data (as PCMFrames) will arrive on the given channel
	// and should be passed meaningfully be the output device (e.g. sent to speakers).
	SetStream(chan<- []int16)
}
