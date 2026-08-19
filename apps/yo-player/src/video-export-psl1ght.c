// PSL1GHT does not currently expose the firmware Video Export utility.
// Preserve completed files in an explicit app-owned directory instead of
// pretending they were inserted into the XMB media database.

#include "video-export.h"
#include "vfs.h"
#include "dbg.h"

#define DOWNLOAD_DIR "/dev_hdd0/tmp/yo-player/downloads"

void initVideoExport(void)
{
   makeDir(DOWNLOAD_DIR);
   logInfo("[export] PSL1GHT mode: downloads stay in %s (XMB database export unavailable)\n",
           DOWNLOAD_DIR);
}

void shutdownVideoExport(void) { }
void updateVideoExport(void) { }

void queueVideoExport(const char *path, const char *title)
{
   (void)title;
   if (path && path[0])
      logInfo("[export] completed file retained at %s\n", path);
}

const char *getVideoStagingDir(void) { return DOWNLOAD_DIR; }
