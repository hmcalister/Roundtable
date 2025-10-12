package audio

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hmcalister/roundtable/internal/audio/device"
	"github.com/hraban/opus"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

const (
	OPUS_SAMPLE_DURATION     = 20 * time.Millisecond
	encoderDecoderBufferSize = 8192
)

var (
	RTPCodecCapability = webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeOpus,
	}
)

// A singleton manager for Audio IO.
// Holds reference to a input device (e.g. microphone), an output device (e.g. speakers)
// and handles fan-in / fan-out of audio inputs and outputs (giving a copy of audio inputs to each peer)
//
// Also handles the encoding and decoding of raw audio data before transmission.
// Currently, only OPUS is supported for encoding and decoding, with
// sample rate of 48000HZ and mono channels.
type AudioManager struct {
	logger *slog.Logger

	// The device to get audio inputs from
	audioInputDevice device.AudioInputDevice

	inputListenersMutex sync.RWMutex
	// a list of listeners for new input data.
	// Effectively peers on the network.
	inputListeners []InputListener

	// The device to send audio outputs to
	audioOutputDevice device.AudioOutputDevice

	// Encoder handles encoding raw PCM frames to OPUS frames
	encoder *opus.Encoder
}

func NewAudioManager(
	audioInputDevice device.AudioInputDevice,
	audioOutputDevice device.AudioOutputDevice,
	logger *slog.Logger,
) (*AudioManager, error) {
	if logger == nil {
		logger = slog.Default()
	}

	encoder, err := opus.NewEncoder(
		audioInputDevice.SampleRate(),
		audioInputDevice.NumChannels(),
		opus.Application(opus.AppVoIP),
	)
	if err != nil {
		logger.Error("error when creating OPUS encoder", "err", err)
	}

	manager := &AudioManager{
		logger:            logger,
		audioInputDevice:  audioInputDevice,
		inputListeners:    make([]InputListener, 0),
		audioOutputDevice: audioOutputDevice,
		encoder:           encoder,
	}

	go manager.handleAudioInput()

	return manager, nil
}

// Fan-out the audio input stream data to all listeners
func (manager *AudioManager) handleAudioInput() {
	go func() {
		encodedBuffer := make([]byte, encoderDecoderBufferSize)
		inputStream := manager.audioInputDevice.GetStream()

		for inputData := range inputStream {
			numEncodedBytes, err := manager.encoder.Encode(inputData, encodedBuffer)
			if err != nil {
				manager.logger.Error("failed to encode raw input data", "err", err)
				continue
			}
			sample := media.Sample{
				Data:     encodedBuffer[:numEncodedBytes],
				Duration: OPUS_SAMPLE_DURATION,
			}

			manager.inputListenersMutex.Lock()
			// TODO: In a naive world, one channel blocking here will cause all channels to block.
			// Current select approach drops data to listeners who can't accept it... is that fine?
			for _, listener := range manager.inputListeners {
				select {
				case listener.dataChannel <- sample:
				default:
					// If listener does not accept immediately, drop
					// This means they miss some data... okay?
				}
			}
			manager.inputListenersMutex.Unlock()
		}
	}()
}

func (manager *AudioManager) AddInputListener() (<-chan media.Sample, context.CancelFunc) {
	manager.inputListenersMutex.Lock()
	defer manager.inputListenersMutex.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	dataChannel := make(chan media.Sample)
	newListener := InputListener{
		uuid:        uuid.New(),
		dataChannel: dataChannel,
	}
	manager.inputListeners = append(manager.inputListeners, newListener)

	// When the context is canceled, traverse the listener list looking for this listener
	// and remove it by splicing in the last listener and shortening the list.
	go func() {
		<-ctx.Done()
		manager.inputListenersMutex.Lock()
		defer manager.inputListenersMutex.Unlock()

		close(dataChannel)
		numListeners := len(manager.inputListeners)
		for i, listener := range manager.inputListeners {
			if listener.uuid == newListener.uuid {
				manager.inputListeners[i] = manager.inputListeners[numListeners-1]
				manager.inputListeners = manager.inputListeners[:numListeners-1]
				return
			}
		}
	}()

	return dataChannel, cancel
}

// Add an output source, something that can play audio on this client.
// Outout sources can play audio by sending OPUS encoded data along the returned channel.
func (manager *AudioManager) AddOPUSOutputSource(sampleRate int, numChannels int) chan<- []byte {
	dataChannel := make(chan []byte)
	decoder, err := opus.NewDecoder(sampleRate, numChannels)
	if err != nil {
		manager.logger.Error("error while constructing decoder", "err", err)
	}
	go func() {
		decodedBuffer := make([]int16, encoderDecoderBufferSize)

		// When dataChannel is closed, we can stop listening on this loop
		for incomingData := range dataChannel {
			numDecodedBytes, err := decoder.Decode(incomingData, decodedBuffer)
			if err != nil {
				slog.Error("failed to decode incoming audio", "err", err)
				continue
			}
			slog.Debug("decoded incoming audio", "numDecodedBytes", numDecodedBytes)
		}
	}()

	return dataChannel
}
