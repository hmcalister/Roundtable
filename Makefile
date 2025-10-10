microphone: microphone.cpp
	g++ -o microphone microphone.cpp RtAudio.cpp -D__UNIX_JACK__ -ljack -lpthread -lm

listAudioDevices: listAudioDevices.cpp
	g++ -o listAudioDevices listAudioDevices.cpp RtAudio.cpp -D__UNIX_JACK__ -ljack -lpthread
