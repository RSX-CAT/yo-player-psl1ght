#pragma once

#include <rsx/rsx.h>
#include <rsx/gcm_sys.h>

typedef gcmConfiguration CellGcmConfig;
typedef gcmSurface CellGcmSurface;
typedef gcmTexture CellGcmTexture;

#define antialias antiAlias
#define memoryFrequency memoryFreq
#define coreFrequency   coreFreq

#define CELL_GCM_TRUE  GCM_TRUE
#define CELL_GCM_FALSE GCM_FALSE
#define CELL_GCM_LOCATION_LOCAL GCM_LOCATION_RSX
#define CELL_GCM_LOCATION_MAIN  GCM_LOCATION_CELL
#define CELL_GCM_SURFACE_A8R8G8B8 GCM_SURFACE_A8R8G8B8
#define CELL_GCM_SURFACE_TARGET_0  GCM_SURFACE_TARGET_0
#define CELL_GCM_SURFACE_Z16       GCM_SURFACE_ZETA_Z16
#define CELL_GCM_SURFACE_PITCH     GCM_SURFACE_TYPE_LINEAR
#define CELL_GCM_SURFACE_CENTER_1  GCM_SURFACE_CENTER_1
#define CELL_GCM_ZCULL_ALIGN_WIDTH  GCM_ZCULL_ALIGN_WIDTH
#define CELL_GCM_ZCULL_ALIGN_HEIGHT GCM_ZCULL_ALIGN_HEIGHT
#define CELL_GCM_TILE_ALIGN_OFFSET  GCM_TILE_ALIGN_OFFSET
#define CELL_GCM_COMPMODE_DISABLED  GCM_COMPMODE_DISABLED
#define CELL_GCM_DISPLAY_VSYNC      GCM_FLIP_VSYNC
#define CELL_GCM_DISPLAY_HSYNC      GCM_FLIP_HSYNC

#define CELL_GCM_COLOR_MASK_R GCM_COLOR_MASK_R
#define CELL_GCM_COLOR_MASK_G GCM_COLOR_MASK_G
#define CELL_GCM_COLOR_MASK_B GCM_COLOR_MASK_B
#define CELL_GCM_COLOR_MASK_A GCM_COLOR_MASK_A
#define CELL_GCM_CLEAR_R GCM_CLEAR_R
#define CELL_GCM_CLEAR_G GCM_CLEAR_G
#define CELL_GCM_CLEAR_B GCM_CLEAR_B
#define CELL_GCM_CLEAR_A GCM_CLEAR_A

#define CELL_GCM_TEXTURE_A8R8G8B8 GCM_TEXTURE_FORMAT_A8R8G8B8
#define CELL_GCM_TEXTURE_B8       GCM_TEXTURE_FORMAT_B8
#define CELL_GCM_TEXTURE_LN       GCM_TEXTURE_FORMAT_LIN
#define CELL_GCM_TEXTURE_NR       GCM_TEXTURE_FORMAT_NRM
#define CELL_GCM_TEXTURE_DIMENSION_2 GCM_TEXTURE_DIMS_2D
#define CELL_GCM_TEXTURE_REMAP_MODE GCM_TEXTURE_REMAP_MODE
#define CELL_GCM_REMAP_MODE         GCM_TEXTURE_REMAP_MODE
#define CELL_GCM_TEXTURE_REMAP_ORDER_XYXY GCM_TEXTURE_REMAP_ORDER_XYXY
#define CELL_GCM_TEXTURE_REMAP_REMAP GCM_TEXTURE_REMAP_TYPE_REMAP
#define CELL_GCM_TEXTURE_REMAP_FROM_A GCM_TEXTURE_REMAP_COLOR_A
#define CELL_GCM_TEXTURE_REMAP_FROM_R GCM_TEXTURE_REMAP_COLOR_R
#define CELL_GCM_TEXTURE_REMAP_FROM_G GCM_TEXTURE_REMAP_COLOR_G
#define CELL_GCM_TEXTURE_REMAP_FROM_B GCM_TEXTURE_REMAP_COLOR_B
#define CELL_GCM_TEXTURE_MAX_ANISO_1 GCM_TEXTURE_MAX_ANISO_1
#define CELL_GCM_TEXTURE_NEAREST GCM_TEXTURE_NEAREST
#define CELL_GCM_TEXTURE_LINEAR  GCM_TEXTURE_LINEAR
#define CELL_GCM_TEXTURE_CONVOLUTION_QUINCUNX GCM_TEXTURE_CONVOLUTION_QUINCUNX
#define CELL_GCM_TEXTURE_CLAMP_TO_EDGE GCM_TEXTURE_CLAMP_TO_EDGE
#define CELL_GCM_TEXTURE_UNSIGNED_REMAP_NORMAL GCM_TEXTURE_UNSIGNED_REMAP_NORMAL
#define CELL_GCM_TEXTURE_ZFUNC_LESS GCM_TEXTURE_ZFUNC_LESS
#define CELL_GCM_INVALIDATE_TEXTURE GCM_INVALIDATE_TEXTURE

#define CELL_GCM_VERTEX_F  GCM_VERTEX_DATA_TYPE_F32
#define CELL_GCM_VERTEX_UB GCM_VERTEX_DATA_TYPE_U8
#define CELL_GCM_PRIMITIVE_TRIANGLES GCM_TYPE_TRIANGLES
#define CELL_GCM_SRC_ALPHA GCM_SRC_ALPHA
#define CELL_GCM_ONE_MINUS_SRC_ALPHA GCM_ONE_MINUS_SRC_ALPHA

static inline uint32_t pslGcmAlign(uint32_t alignment, uint32_t value)
{
   return (value + alignment - 1u) & ~(alignment - 1u);
}

static inline void pslSetVertexProgram(gcmContextData *context,
                                       const rsxVertexProgram *program,
                                       const void *ucode)
{
   rsxLoadVertexProgram(context, program, ucode);
   rsxSetVertexAttribOutputMask(context, program->output_mask);
}

static inline void pslSetFragmentProgram(gcmContextData *context,
                                         const rsxFragmentProgram *program,
                                         uint32_t offset)
{
   rsxLoadFragmentProgramLocation(context, program, offset, GCM_LOCATION_RSX);
}

/* The Sony SDK places the vertex-buffer offset last.  PSL1GHT follows the
 * hardware command order and places it before stride, so this cannot be a
 * direct preprocessor alias. */
static inline void pslSetVertexDataArray(gcmContextData *context,
                                         uint8_t index,
                                         uint16_t frequency,
                                         uint8_t stride,
                                         uint8_t size,
                                         uint8_t type,
                                         uint8_t location,
                                         uint32_t offset)
{
   rsxBindVertexArrayAttrib(context, index, frequency, offset, stride, size,
                            type, location);
}

#define cellGcmAlign                pslGcmAlign
#define cellGcmGetTiledPitchSize    gcmGetTiledPitchSize
#define cellGcmGetConfiguration     gcmGetConfiguration
#define cellGcmAddressToOffset      rsxAddressToOffset
#define cellGcmSetSurface           rsxSetSurface
#define cellGcmSetViewport          rsxSetViewport
#define cellGcmSetScissor           rsxSetScissor
#define cellGcmSetTileInfo          gcmSetTileInfo
#define cellGcmBindTile             gcmBindTile
#define cellGcmSetDisplayBuffer     gcmSetDisplayBuffer
#define cellGcmSetFlipHandler       gcmSetFlipHandler
#define cellGcmSetFlipMode          gcmSetFlipMode
#define cellGcmSetWaitFlip          gcmSetWaitFlip
#define cellGcmSetFlip              gcmSetFlip
#define cellGcmFlush                rsxFlushBuffer
#define cellGcmFinish               rsxFinish
#define cellGcmSetColorMask         rsxSetColorMask
#define cellGcmSetColorMaskMrt      rsxSetColorMaskMrt
#define cellGcmSetDepthTestEnable   rsxSetDepthTestEnable
#define cellGcmSetFragmentProgramGammaEnable rsxSetFragmentProgramGammaEnable
#define cellGcmSetClearColor        rsxSetClearColor
#define cellGcmSetClearSurface      rsxClearSurface
#define cellGcmSetVertexProgram     pslSetVertexProgram
#define cellGcmSetFragmentProgram   pslSetFragmentProgram
#define cellGcmSetVertexDataArray   pslSetVertexDataArray
#define cellGcmSetTexture           rsxLoadTexture
#define cellGcmSetTextureControl    rsxTextureControl
#define cellGcmSetTextureAddress    rsxTextureWrapMode
#define cellGcmSetTextureFilter     rsxTextureFilter
#define cellGcmSetInvalidateTextureCache rsxInvalidateTextureCache
#define cellGcmSetCullFaceEnable    rsxSetCullFaceEnable
#define cellGcmSetBlendEnable       rsxSetBlendEnable
#define cellGcmSetBlendFunc         rsxSetBlendFunc
#define cellGcmSetDrawArrays        rsxDrawVertexArray
#define cellGcmMapMainMemory        gcmMapMainMemory
#define cellGcmUnmapEaIoAddress     gcmUnmapEaIoAddress
