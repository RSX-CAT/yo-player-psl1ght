#pragma once

// decode-aac - AAC-LC/HE-AAC decoding behind one interface.  The official-SDK build uses cellAdec;
// the PSL1GHT build uses FAAD2 because PSL1GHT has no public ADEC binding. Feed it ADTS frames and
// pull decoded audio back as interleaved stereo float32 (multi-channel input is downmixed by the
// decoder, mono is duplicated to both channels).

#include <stdint.h>

typedef struct AacDecoder AacDecoder;

// one decoded AAC frame is 1024 samples (2048 when SBR doubles it)
#define AAC_MAX_FRAME_SAMPLES 2048
#define ADTS_HEADER_BYTES     7

AacDecoder *createAacDecoder(void);   // NULL on failure; rate/channels come from the ADTS headers

// feeds one complete ADTS frame (header + payload). pts is nanoseconds and is round-tripped to the
// matching PCM. returns 0 when accepted, 1 when the decoder's queue is full (pull PCM, then retry
// the same frame), -1 on error. `data` must stay untouched until the backlog says it was consumed.
int decodeAuAac(AacDecoder *decoder, const uint8_t *data, int size, uint64_t pts);

// ADTS frames fed but not yet consumed. A caller with N rotating frame buffers may safely build a
// new frame while the backlog is below N.
int getAuBacklogAac(const AacDecoder *decoder);

// retrieves one decoded frame as interleaved stereo float32 into `stereoOut` (must hold
// AAC_MAX_FRAME_SAMPLES * 2 floats). returns 1 and sets *outFrames/*outRate/*outPts when audio was
// ready, 0 if none pending (or a concealed/empty frame was discarded), -1 on error.
int getPcmAac(AacDecoder *decoder, float *stereoOut, int *outFrames, int *outRate, uint64_t *outPts);

// flushes the decode pipeline for a seek: ends the sequence (discarding in-flight frames) and
// starts a fresh one on the same handle. 0 ok, -1 on error.
int resetAacDecoder(AacDecoder *decoder);

void destroyAacDecoder(AacDecoder *decoder);

// writes the ADTS header for one raw AAC frame (MKV stores frames headerless). AAC-LC profile is
// assumed; SBR extensions are in-band and auto-detected by the decoder. 0 ok, -1 if the rate or
// channel count can't be expressed in ADTS.
int buildAdtsHeader(uint8_t out[ADTS_HEADER_BYTES], int payloadBytes, int rate, int channels);
