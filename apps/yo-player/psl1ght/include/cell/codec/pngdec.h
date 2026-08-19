#pragma once

#include <pngdec/pngdec.h>

typedef s32 CellPngDecMainHandle;
typedef s32 CellPngDecSubHandle;
typedef pngDecThreadInParam CellPngDecThreadInParam;
typedef pngDecThreadOutParam CellPngDecThreadOutParam;
typedef pngDecSource CellPngDecSrc;
typedef pngDecInfo CellPngDecInfo;
typedef pngDecDataInfo CellPngDecDataOutInfo;
typedef pngDecInParam CellPngDecInParam;
typedef pngDecOutParam CellPngDecOutParam;
typedef pngDecDataCtrlParam CellPngDecDataCtrlParam;
typedef pngDecOpnInfo CellPngDecOpnInfo;

#define CELL_PNGDEC_SPU_THREAD_DISABLE PNGDEC_SPU_THREAD_DISABLE
#define CELL_PNGDEC_BUFFER             PNGDEC_BUFFER
#define CELL_PNGDEC_GRAYSCALE_ALPHA    PNGDEC_GRAYSCALE_ALPHA
#define CELL_PNGDEC_RGBA               PNGDEC_RGBA
#define CELL_PNGDEC_ARGB               PNGDEC_ARGB
#define CELL_PNGDEC_TOP_TO_BOTTOM      PNGDEC_TOP_TO_BOTTOM
#define CELL_PNGDEC_1BYTE_PER_1PIXEL   PNGDEC_1BYTE_PER_1PIXEL
#define CELL_PNGDEC_STREAM_ALPHA       PNGDEC_STREAM_ALPHA
#define CELL_PNGDEC_FIX_ALPHA          PNGDEC_FIX_ALPHA
#define CELL_PNGDEC_DEC_STATUS_FINISH  PNGDEC_STATUS_FINISH

#define spuThreadEnable   spu_enable
#define ppuThreadPriority ppu_prio
#define spuThreadPriority spu_prio
#define cbCtrlMallocFunc  malloc_func
#define cbCtrlMallocArg   malloc_arg
#define cbCtrlFreeFunc    free_func
#define cbCtrlFreeArg     free_arg
#define srcSelect         stream_sel
#define fileName          file_name
#define fileOffset        file_offset
#define fileSize          file_size
#define streamPtr         stream_ptr
#define streamSize        stream_size
#define imageWidth        width
#define imageHeight       height
#define colorSpace        color_space
#define commandPtr        cmd_ptr
#define outputMode        output_mode
#define outputColorSpace  color_space
#define outputBitDepth    bit_depth
#define outputPackFlag    pack_flag
#define outputAlphaSelect alpha_select
#define outputColorAlpha  alpha
#define outputWidth       width
#define outputHeight      height
#define outputBytesPerLine output_bytes_per_line

#define cellPngDecCreate       pngDecCreate
#define cellPngDecOpen         pngDecOpen
#define cellPngDecReadHeader   pngDecReadHeader
#define cellPngDecSetParameter pngDecSetParameter
#define cellPngDecDecodeData   pngDecDecodeData
#define cellPngDecClose        pngDecClose
#define cellPngDecDestroy      pngDecDestroy
