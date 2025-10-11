# Roundtable

### A Peer-to-Peer VoIP Application

---

Roundtable is under active development. In its current state, the application is not ready for deployment. The structure of the program may change drastically during this phase of development.

# Plan

The aim of Roundtable is to allow for Peer-to-Peer (P2P), and hence low-latency, communication of voice. This requires several key modules to be implemented:

### Getting Voice Data
- Obtain signal/data from the microphone/hardware device
    - This should have cross-platform support!
    - Check for libraries to make this aspect easier
    - Testing: it may be possible to spoof the hardware device for testing purposes... for later stages of Ingress, it should be possible to have a static audio file that is loaded for testing.

- Transform data into a serializable format to send to peers
    - This may depend on the P2P networking library used, and what formats it supports
    - Testing: this aspect should be easy enough to test with static files or perhaps fuzzing

- Send data to peers
    - Using what ever P2P library is decided upon, send the serialized data to all peers
    - Realistically this is going to be done in a streaming-like manner to avoid waiting
    - Sending should be handled by a distinct thread per peer to minimize latency
    - The P2P library may require buffering or other logic, which could mean additional work in packaging the data before sending
    - Testing: it may be possible to spoof the networking interface here, to avoid actually setting up a connection/peer

### Playing Peer Data
- Read data from each peer
    - Just like sending data to each peer, this should be threaded to minimize latency, then any results should be fed into one parent thread for consolidation
    - Testing: spoofing the network may be possible again... consider what happens when there are many peers?

- Deserialize peer data into a useful format for playback
    - The source format is going to be entirely determined by the P2P library, but the final format could be any number of things...
    - Depending on the playback method, the format may be more restrictive
    - If the network delivers data in a stream, is it possible to play the most recent fragment and still achieve a useful user experience?
    - Testing: static files and fuzzing should be sufficient for testing transformations

- Combine all peer fragments into one stream ready for playback, apply any filters
    - If two peers send data at once, they should be played simultaneously. Combining two distinct data streams should be investigated
    - It may be possible to apply filters (e.g. noise reduction) to the data stream before playing it to the user
    - Testing: combining signals may be easy to test with static files, same for filters

- Send final data stream to audio output hardware
    - Just like input, this should be cross-platform
    - Testing: TBD

### Networking
P2P Networking is non-trivial... consider WebRTC as a foundation. Alternatively, something based on ConnectRPC may be applicable.

Testing throughout this section is going to be extremely difficult!

- Determine public facing IP
    - P2P requires punching through NAT to allow connections, since almost all users will not have a static public IP
    - STUN servers seem to allow for this, some are publicly available, and it doesn't seem to be a privacy issue to use these

- Set up connections to peers
    - Once a user has started the application, how can they connect to another user? 
    - What about to an existing group of users?
    - What about a user in a group adding a new user?
    - Consider security... what if a bad actor tries to connect? How should a connection be verified / accepted?

### Other Factors

- Config
    - A nice configuration method would be ideal, such as selecting STUN servers, setting any security for incoming connections, and so on

- Security
    - Require a password or token for incoming connections?
    - Lock a group to prevent any new connections?
    - Rate limit an IP address to avoid a bad-actor hammering a user?

- Various user-experience methods
    - Mute, deafen, etc...
    - Per-user equalizer, i.e. set a volume level per user
    - Notification sounds for a new connection, a lost connection, etc.

- User interface
    - Interactions beyond simple audio-in, audio-out appear to be complex enough to warrant a user interface
    - Something terminal based, which could be bound to underlying data models with ease, could be useful
    - A cross-platform GUI could be more difficult to implement correctly

### Building of RtAudio

Currently RtAudio is built with make:

- `make windows` for windows target
- `make linux` for linux target

The only difference is the libaries linked and the audio backend defined e.g.  `-D__UNIX_JACK__`  for jack audio on linux

```Make

microphone: microphone.cpp
	g++ -o microphone microphone.cpp RtAudio.cpp -D__UNIX_JACK__ -lpthread -lm -ljack 


WINLIBS :=  -lole32 -lwinmm -lksuser -lmfplat -lmfuuid -lwmcodecdspuuid
microphoneWindows: microphone.cpp
	g++ -o microphoneWindows microphone.cpp RtAudio.cpp -D__WINDOWS_WASAPI__ $(WINLIBS)
```

This also builds a binary for listing audio devices, which may be useful for debugging and showing device related requirements e.g. sample rate.

The `./microphone` just records the system default mic for provided amount of seconds and writes it to a wav file for testing proof of concept.
The usage:

```
usage: record N fs <duration> <device> <channelOffset>
    N                     = number of channels
    fs                    = the sample rate
    duration              = optional time in seconds to record (default = 2.0)
    channelOffset         = an optional channel offset on the "
    device       (default = 0)
  
```

