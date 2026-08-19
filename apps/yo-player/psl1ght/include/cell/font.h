#pragma once

#include <string.h>
#include <font/font.h>

typedef fontMemoryInterface CellFontMemoryInterface;
typedef fontEntry CellFontEntry;
typedef fontConfig CellFontConfig;
typedef fontLibrary CellFontLibrary;
typedef fontType CellFontType;
typedef font CellFont;
typedef fontRenderer CellFontRenderer;
typedef fontRendererConfig CellFontRendererConfig;
typedef fontHorizontalLayout CellFontHorizontalLayout;
typedef fontGlyphMetrics CellFontGlyphMetrics;
typedef fontKerning CellFontKerning;
typedef fontRenderSurface CellFontRenderSurface;
typedef fontImageTransInfo CellFontImageTransInfo;

#define CELL_FONT_MAP_UNICODE FONT_MAP_UNICODE
#define CELL_FONT_TYPE_DEFAULT_GOTHIC_LATIN_SET FONT_TYPE_DEFAULT_GOTHIC_LATIN_SET
#define CELL_FONT_TYPE_DEFAULT_GOTHIC_JP_SET    FONT_TYPE_DEFAULT_GOTHIC_JP_SET
#define CELL_FONT_TYPE_DEFAULT_SANS_SERIF       FONT_TYPE_DEFAULT_SANS_SERIF
#define CELL_FONT_TYPE_DEFAULT_SERIF            FONT_TYPE_DEFAULT_SERIF

#define FileCache      fileCache
#define userFontEntrys userFontEntries
#define Horizontal     horizontal
#define Vertical       vertical
#define Image          image
#define Surface        surface

/* Current PSL1GHT exports this symbol but its public header contains the
 * historical fontontSetScalePixel spelling.  Declare the exported spelling
 * locally instead of depending on an implicit C function declaration. */
s32 fontSetScalePixel(font *fontHandle, f32 width, f32 height);

static inline void CellFontConfig_initialize(CellFontConfig *config)
{
   fontConfig_initialize(config);
}

static inline void CellFontRendererConfig_initialize(CellFontRendererConfig *config)
{
   memset(config, 0, sizeof(*config));
}

static inline void CellFontRendererConfig_setAllocateBuffer(
   CellFontRendererConfig *config, uint32_t initialSize, uint32_t maxSize)
{
   config->bufferingPolicy.initSize = initialSize;
   config->bufferingPolicy.maxSize = maxSize;
}

#define cellFontInit                   fontInit
#define cellFontOpenFontset            fontOpenFontset
#define cellFontOpenFont               fontOpenFontFile
#define cellFontOpenFontMemory         fontOpenFontMemory
#define cellFontSetResolutionDpi       fontSetResolutionDpi
#define cellFontSetScalePixel          fontSetScalePixel
#define cellFontGetHorizontalLayout    fontGetHorizontalLayout
#define cellFontGetCharGlyphMetrics    fontGetCharGlyphMetrics
#define cellFontGetKerning             fontGetKerning
#define cellFontCreateRenderer         fontCreateRenderer
#define cellFontBindRenderer           fontBindRenderer
#define cellFontSetupRenderScalePixel  fontSetupRenderScalePixel
#define cellFontSetupRenderEffectSlant fontSetupRenderEffectSlant
#define cellFontRenderSurfaceInit      fontRenderSurfaceInit
#define cellFontRenderSurfaceSetScissor fontRenderSurfaceSetScissor
#define cellFontRenderCharGlyphImage   fontRenderCharGlyphImage
#define cellFontCloseFont              fontCloseFont
#define cellFontDestroyRenderer        fontDestroyRenderer
#define cellFontEndLibrary             fontEndLibrary
#define cellFontEnd                    fontEnd
