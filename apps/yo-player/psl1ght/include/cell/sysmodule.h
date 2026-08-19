#pragma once

#include <sysmodule/sysmodule.h>

#define CELL_SYSMODULE_ERROR_DUPLICATED SYSMODULE_ERR_DUPLICATE
#define CELL_SYSMODULE_NET              SYSMODULE_NET
#define CELL_SYSMODULE_RTC              SYSMODULE_RTC
#define CELL_SYSMODULE_PNGDEC           SYSMODULE_PNGDEC
#define CELL_SYSMODULE_JPGDEC           SYSMODULE_JPGDEC
#define CELL_SYSMODULE_AUDIO            SYSMODULE_AUDIO
#define CELL_SYSMODULE_FONT             SYSMODULE_FONT
#define CELL_SYSMODULE_FONTFT           SYSMODULE_FONTFT
#define CELL_SYSMODULE_FREETYPE         SYSMODULE_FREETYPE
#define CELL_SYSMODULE_VDEC_AVC         SYSMODULE_VDEC_H264

#define cellSysmoduleLoadModule   sysModuleLoad
#define cellSysmoduleUnloadModule sysModuleUnload
