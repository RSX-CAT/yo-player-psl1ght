#pragma once

#include <sysutil/osk.h>

#define CELL_OSKDIALOG_STRING_SIZE 512
#define CELL_OSKDIALOG_INPUT_FIELD_RESULT_OK OSK_OK
#define CELL_OSKDIALOG_PANELMODE_ALPHABET OSK_PANEL_TYPE_ALPHABET
#define CELL_OSKDIALOG_PANELMODE_NUMERAL  OSK_PANEL_TYPE_NUMERAL
#define CELL_OSKDIALOG_PANELMODE_ENGLISH  OSK_PANEL_TYPE_ENGLISH
#define CELL_OSKDIALOG_NO_RETURN          OSK_PROHIBIT_RETURN
#define CELL_OSKDIALOG_10KEY_PANEL        OSK_10KEY_PANEL
#define CELL_OSKDIALOG_FULLKEY_PANEL      OSK_FULLKEY_PANEL
#define SYS_MEMORY_CONTAINER_ID_INVALID   ((sys_mem_container_t)0xffffffffu)

typedef oskCallbackReturnParam CellOskDialogCallbackReturnParam;
typedef oskInputFieldInfo CellOskDialogInputFieldInfo;
typedef oskParam CellOskDialogParam;

#define result               res
#define numCharsResultString len
#define pResultString        str
#define init_text            startText
#define limit_length         maxLength
#define allowOskPanelFlg     allowedPanels
#define prohibitFlgs         prohibitFlags

#define cellOskDialogUnloadAsync         oskUnloadAsync
#define cellOskDialogLoadAsync           oskLoadAsync
#define cellOskDialogAddSupportLanguage  oskAddSupportLanguage
#define cellOskDialogSetKeyLayoutOption  oskSetKeyLayoutOption
