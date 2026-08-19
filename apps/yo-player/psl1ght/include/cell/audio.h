#pragma once

#include <audio/audio.h>

typedef struct CellAudioPortParam {
   uint64_t nChannel;
   uint64_t nBlock;
   uint64_t attr;
   float level;
} CellAudioPortParam;

typedef struct CellAudioPortConfig {
   uint32_t readIndexAddr;
   uint32_t status;
   uint64_t channelCount;
   uint64_t numBlocks;
   uint32_t portSize;
   uint32_t portAddr;
} CellAudioPortConfig;

#define CELL_AUDIO_BLOCK_SAMPLES AUDIO_BLOCK_SAMPLES
#define CELL_AUDIO_PORTATTR_BGM  AUDIO_PORT_INITLEVEL

#define cellAudioInit          audioInit
#define cellAudioQuit          audioQuit
#define cellAudioPortStart     audioPortStart
#define cellAudioPortStop      audioPortStop
#define cellAudioPortClose     audioPortClose

static inline s32 cellAudioPortOpen(CellAudioPortParam *param, uint32_t *port)
{
   audioPortParam native;
   native.numChannels = param->nChannel;
   native.numBlocks = param->nBlock;
   native.attrib = param->attr;
   native.level = param->level;
   return audioPortOpen(&native, port);
}

static inline s32 cellAudioGetPortConfig(uint32_t port, CellAudioPortConfig *config)
{
   return audioGetPortConfig(port, (audioPortConfig *)config);
}
