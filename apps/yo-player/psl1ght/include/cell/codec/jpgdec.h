#pragma once

#include <jpgdec/jpgdec.h>

typedef s32 CellJpgDecMainHandle;
typedef s32 CellJpgDecSubHandle;
typedef jpgDecThreadInParam CellJpgDecThreadInParam;
typedef jpgDecThreadOutParam CellJpgDecThreadOutParam;
typedef jpgDecSource CellJpgDecSrc;
typedef jpgDecInfo CellJpgDecInfo;
typedef jpgDecDataInfo CellJpgDecDataOutInfo;
typedef jpgDecInParam CellJpgDecInParam;
typedef jpgDecOutParam CellJpgDecOutParam;
typedef jpgDecDataCtrlParam CellJpgDecDataCtrlParam;
typedef jpgDecOpnInfo CellJpgDecOpnInfo;

#define CELL_JPGDEC_SPU_THREAD_DISABLE JPGDEC_SPU_THREAD_DISABLE
#define CELL_JPGDEC_BUFFER             JPGDEC_BUFFER
#define CELL_JPGDEC_QUALITY            JPGDEC_QUALITY
#define CELL_JPGDEC_TOP_TO_BOTTOM      JPGDEC_TOP_TO_BOTTOM
#define CELL_JPG_ARGB                  JPGDEC_ARGB
#define CELL_JPGDEC_DEC_STATUS_FINISH  JPGDEC_STATUS_FINISH

// Field aliases shared with the PNG compatibility header are guarded so both
// codec headers may be included by one translation unit.
#ifndef spuThreadEnable
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
#define outputColorAlpha  alpha
#define outputWidth       width
#define outputHeight      height
#define outputBytesPerLine output_bytes_per_line
#endif
#define downScale         down_scale
#define method            quality_mode

#define cellJpgDecCreate       jpgDecCreate
#define cellJpgDecOpen         jpgDecOpen
#define cellJpgDecReadHeader   jpgDecReadHeader
#define cellJpgDecSetParameter jpgDecSetParameter
#define cellJpgDecDecodeData   jpgDecDecodeData
#define cellJpgDecClose        jpgDecClose
#define cellJpgDecDestroy      jpgDecDestroy
