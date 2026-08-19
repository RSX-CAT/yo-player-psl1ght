// PSL1GHT AAC adapter. PSL1GHT has no public ADEC API, so decode ADTS frames
// synchronously with the FAAD2 port supplied by ps3libraries.

#include "decode-aac.h"
#include "dbg.h"

#include <neaacdec.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

struct AacDecoder {
   NeAACDecHandle handle;
   int initialized;
   int pending;
   int frames;
   int rate;
   uint64_t pts;
   int discardedCount;
   float pcm[AAC_MAX_FRAME_SAMPLES * 2];
};

static int configureFromAdts(AacDecoder *decoder, const uint8_t *data, int size)
{
   if (size < ADTS_HEADER_BYTES || data[0] != 0xff || (data[1] & 0xf6) != 0xf0)
      return -1;

   unsigned long rate = 0;
   unsigned char outputChannels = 0;
   /* NeAACDecInit detects and records ADTS framing. NeAACDecInit2 would put
    * FAAD2 into RAW mode, making the same ADTS header look like AAC payload. */
   if (NeAACDecInit(decoder->handle, (unsigned char *)data,
                    (unsigned long)size, &rate, &outputChannels) < 0)
      return -1;
   decoder->initialized = 1;
   return 0;
}

AacDecoder *createAacDecoder(void)
{
   AacDecoder *decoder = (AacDecoder *)calloc(1, sizeof *decoder);
   if (!decoder) return NULL;

   decoder->handle = NeAACDecOpen();
   if (!decoder->handle) { free(decoder); return NULL; }

   NeAACDecConfigurationPtr config = NeAACDecGetCurrentConfiguration(decoder->handle);
   config->outputFormat = FAAD_FMT_16BIT;
   config->downMatrix = 1;
   config->dontUpSampleImplicitSBR = 0;
   if (!NeAACDecSetConfiguration(decoder->handle, config)) {
      NeAACDecClose(decoder->handle);
      free(decoder);
      return NULL;
   }

   logInfo("[decode-aac] opened FAAD2 software decoder\n");
   return decoder;
}

int decodeAuAac(AacDecoder *decoder, const uint8_t *data, int size, uint64_t pts)
{
   if (!decoder || !data || size <= ADTS_HEADER_BYTES) return -1;
   if (decoder->pending) return 1;
   if (!decoder->initialized && configureFromAdts(decoder, data, size) != 0) {
      logError("[decode-aac] invalid/unsupported ADTS configuration\n");
      return -1;
   }

   NeAACDecFrameInfo info;
   memset(&info, 0, sizeof info);
   int16_t *samples = (int16_t *)NeAACDecDecode(decoder->handle, &info,
                                                (unsigned char *)data,
                                                (unsigned long)size);
   if (info.error || !samples || info.samples == 0 || info.channels == 0) {
      decoder->discardedCount++;
      if (decoder->discardedCount == 1 || decoder->discardedCount % 256 == 0)
         logWarn("[decode-aac] discarded frame #%d, error=%u (%s)\n",
                 decoder->discardedCount, info.error,
                 info.error ? NeAACDecGetErrorMessage(info.error) : "empty output");
      return 0;
   }

   int frames = (int)(info.samples / info.channels);
   if (frames > AAC_MAX_FRAME_SAMPLES) frames = AAC_MAX_FRAME_SAMPLES;
   for (int i = 0; i < frames; i++) {
      int16_t left = samples[i * info.channels];
      int16_t right = info.channels > 1 ? samples[i * info.channels + 1] : left;
      decoder->pcm[i * 2] = (float)left / 32768.0f;
      decoder->pcm[i * 2 + 1] = (float)right / 32768.0f;
   }

   decoder->frames = frames;
   decoder->rate = (int)info.samplerate;
   decoder->pts = pts;
   decoder->pending = 1;
   return 0;
}

int getAuBacklogAac(const AacDecoder *decoder)
{
   return decoder && decoder->pending ? 1 : 0;
}

int getPcmAac(AacDecoder *decoder, float *stereoOut, int *outFrames,
              int *outRate, uint64_t *outPts)
{
   if (!decoder || !stereoOut || !outFrames || !outRate || !outPts) return -1;
   if (!decoder->pending) return 0;
   memcpy(stereoOut, decoder->pcm, (size_t)decoder->frames * 2 * sizeof(float));
   *outFrames = decoder->frames;
   *outRate = decoder->rate;
   *outPts = decoder->pts;
   decoder->pending = 0;
   return 1;
}

int resetAacDecoder(AacDecoder *decoder)
{
   if (!decoder || !decoder->handle) return -1;
   NeAACDecPostSeekReset(decoder->handle, 0);
   decoder->pending = 0;
   return 0;
}

void destroyAacDecoder(AacDecoder *decoder)
{
   if (!decoder) return;
   if (decoder->handle) NeAACDecClose(decoder->handle);
   free(decoder);
}

int buildAdtsHeader(uint8_t out[ADTS_HEADER_BYTES], int payloadBytes,
                    int rate, int channels)
{
   static const int rates[] = { 96000, 88200, 64000, 48000, 44100, 32000,
                                24000, 22050, 16000, 12000, 11025, 8000, 7350 };
   int frequencyIndex = -1;
   for (int i = 0; i < (int)(sizeof rates / sizeof rates[0]); i++)
      if (rates[i] == rate) { frequencyIndex = i; break; }
   if (frequencyIndex < 0 || channels < 1 || channels > 6 || payloadBytes <= 0)
      return -1;

   int frameLength = payloadBytes + ADTS_HEADER_BYTES;
   out[0] = 0xff; out[1] = 0xf1;
   out[2] = (uint8_t)(0x40 | (frequencyIndex << 2) | (channels >> 2));
   out[3] = (uint8_t)(((channels & 3) << 6) | ((frameLength >> 11) & 3));
   out[4] = (uint8_t)(frameLength >> 3);
   out[5] = (uint8_t)(((frameLength & 7) << 5) | 0x1f);
   out[6] = 0xfc;
   return 0;
}
