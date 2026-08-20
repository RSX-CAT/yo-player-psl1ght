#pragma once

#include <font/fontFT.h>

typedef fontLibraryConfigFT CellFontLibraryConfigFT;

/* Match the revision used by the known-good official SDK 4.75 EBOOT. */
#define YO_PLAYER_CELL_FONT_FT_REVISION 0x42ULL
extern s32 fontInitLibraryFreeTypeWithRevision(
   u64 revisionFlags,
   fontLibraryConfigFT *config,
   const fontLibrary *ATTRIBUTE_PRXPTR *lib);

static inline s32 yoPlayerFontInitLibraryFreeType(
   CellFontLibraryConfigFT *config, const CellFontLibrary **lib)
{
   const fontLibrary *result ATTRIBUTE_PRXPTR = NULL;
   if (config == NULL || lib == NULL) return (s32)0x80540002;

   s32 rc = fontInitLibraryFreeTypeWithRevision(
      YO_PLAYER_CELL_FONT_FT_REVISION, config, &result);
   *lib = rc == 0 ? result : NULL;
   return rc;
}

#define MemoryIF memoryIF
#define Object   object
#define Malloc   malloc_func
#define Free     free_func
#define Realloc  realloc_func
#define Calloc   calloc_func

static inline void CellFontLibraryConfigFT_initialize(CellFontLibraryConfigFT *config)
{
   fontLibraryConfigFT_initialize(config);
}

#define cellFontInitLibraryFreeType yoPlayerFontInitLibraryFreeType
