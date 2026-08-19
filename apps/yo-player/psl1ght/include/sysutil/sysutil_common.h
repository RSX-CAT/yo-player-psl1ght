#pragma once

#include <sysutil/sysutil.h>

#define CELL_SYSUTIL_REQUEST_EXITGAME  SYSUTIL_EXIT_GAME
#define CELL_SYSUTIL_SYSTEM_MENU_OPEN  SYSUTIL_MENU_OPEN
#define CELL_SYSUTIL_SYSTEM_MENU_CLOSE SYSUTIL_MENU_CLOSE
#define CELL_SYSUTIL_OSKDIALOG_FINISHED SYSUTIL_OSK_DONE
#define CELL_SYSUTIL_OSKDIALOG_UNLOADED SYSUTIL_OSK_UNLOADED

#define cellSysutilRegisterCallback   sysUtilRegisterCallback
#define cellSysutilUnregisterCallback sysUtilUnregisterCallback
#define cellSysutilCheckCallback      sysUtilCheckCallback
