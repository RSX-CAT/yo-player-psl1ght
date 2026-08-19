#pragma once

#include <sys/file.h>
#include <lv2/sysfs.h>

typedef sysFSStat CellFsStat;
typedef sysFSDirent CellFsDirent;

#define CELL_FS_SUCCEEDED 0
#define CELL_FS_S_IFDIR   0040000
#define CELL_FS_O_RDONLY  SYS_O_RDONLY
#define CELL_FS_O_WRONLY  SYS_O_WRONLY
#define CELL_FS_O_RDWR    SYS_O_RDWR
#define CELL_FS_O_CREAT   SYS_O_CREAT
#define CELL_FS_O_TRUNC   SYS_O_TRUNC
#define CELL_FS_O_APPEND  SYS_O_APPEND
#define CELL_FS_SEEK_SET  0
#define CELL_FS_SEEK_CUR  1
#define CELL_FS_SEEK_END  2
#define CELL_FS_TYPE_UNKNOWN   0
#define CELL_FS_TYPE_DIRECTORY 1
#define CELL_FS_TYPE_REGULAR   2
#define CELL_FS_TYPE_SYMLINK   3

static inline s32 pslCellFsOpen(const char *path, s32 flags, s32 *descriptor,
                                const void *arg, uint64_t argSize)
{
   return sysFsOpen(path, flags, descriptor, arg, argSize);
}

static inline s32 pslCellFsMkdir(const char *path, s32 mode)
{
   s32 result = sysFsMkdir(path, mode);
   if (result == 0) return 0;
   sysFSStat st;
   return sysFsStat(path, &st) == 0 && (st.st_mode & CELL_FS_S_IFDIR) ? 0 : result;
}

static inline s32 pslCellFsUnlink(const char *path)
{
   s32 result = sysFsUnlink(path);
   if (result == 0) return 0;
   sysFSStat st;
   return sysFsStat(path, &st) != 0 ? 0 : result;
}

#define cellFsStat        sysFsStat
#define cellFsRename      sysLv2FsRename
#define cellFsMkdir       pslCellFsMkdir
#define cellFsUnlink      pslCellFsUnlink
#define cellFsRmdir       sysFsRmdir
#define cellFsGetFreeSize sysFsGetFreeSize
#define cellFsOpendir     sysFsOpendir
#define cellFsReaddir     sysFsReaddir
#define cellFsClosedir    sysFsClosedir
#define cellFsOpen        pslCellFsOpen
#define cellFsRead        sysFsRead
#define cellFsWrite       sysFsWrite
#define cellFsLseek       sysFsLseek
#define cellFsClose       sysFsClose
