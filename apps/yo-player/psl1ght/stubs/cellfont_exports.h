#ifndef YO_PLAYER_CELLFONT_EXPORTS_H
#define YO_PLAYER_CELLFONT_EXPORTS_H

/* Only the cellFont entry points used by Yo! Player.  In particular, do not
 * import cellFontGetRevisionFlags: libfontFT must bind that dependency to the
 * firmware libfont export rather than to an app-side unresolved stub. */
EXPORT(fontCreateRenderer, 0x042e74e3);
EXPORT(fontGetKerning, 0x06be743d);
EXPORT(fontOpenFontFile, 0x0a7306a4);
EXPORT(fontGetHorizontalLayout, 0x1387c45c);
EXPORT(fontDestroyRenderer, 0x21ebb248);
EXPORT(fontSetupRenderScalePixel, 0x227e1e3c);
EXPORT(fontSetScalePixel, 0x297f0e93);
EXPORT(fontEndLibrary, 0x40d40544);
EXPORT(fontBindRenderer, 0x66a23100);
EXPORT(fontSetupRenderEffectSlant, 0x78d05e08);
EXPORT(fontEnd, 0x7ab47f7e);
EXPORT(fontRenderCharGlyphImage, 0x88be4799);
EXPORT(fontRenderSurfaceInit, 0x90b9465e);
EXPORT(fontOpenFontMemory, 0x9e19072b);
EXPORT(fontOpenFontset, 0xa885cc9b);
EXPORT(fontCloseFont, 0xb276f1f6);
EXPORT(fontRenderSurfaceSetScissor, 0xb422b005);
EXPORT(fontGetCharGlyphMetrics, 0xd8eaee9f);
EXPORT(fontInitializeWithRevision, 0xf03dcc29);
EXPORT(fontSetResolutionDpi, 0xfb3341ba);

#endif
