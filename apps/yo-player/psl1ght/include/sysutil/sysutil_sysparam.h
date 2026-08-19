#pragma once

#include <sysutil/video.h>

typedef videoState CellVideoOutState;
typedef videoResolution CellVideoOutResolution;
typedef videoConfiguration CellVideoOutConfiguration;

#define CELL_VIDEO_OUT_PRIMARY VIDEO_PRIMARY
#define CELL_VIDEO_OUT_BUFFER_COLOR_FORMAT_X8R8G8B8 VIDEO_BUFFER_FORMAT_XRGB
#define resolutionId resolution

#define cellVideoOutGetState      videoGetState
#define cellVideoOutGetResolution videoGetResolution
#define cellVideoOutConfigure     videoConfigure
