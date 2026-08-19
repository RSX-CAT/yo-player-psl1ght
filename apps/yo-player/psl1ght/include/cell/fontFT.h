#pragma once

#include <font/fontFT.h>

typedef fontLibraryConfigFT CellFontLibraryConfigFT;

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

#define cellFontInitLibraryFreeType fontInitLibraryFreeType
