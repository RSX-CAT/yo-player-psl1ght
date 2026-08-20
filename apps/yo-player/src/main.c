// yo-player - a native PS3 YouTube client (in progress).
//
// Boilerplate mirrors file-manager (the proven full app): RTC + network up
// front, bridge logging, VSYNC-ON gfx, then the frame loop. It boots into the
// home screen (trending categories); searching / channels / playback stream via
// simple-lib-av. Bring-up diagnostics go to the bridge log (dbg.h).

#include <sys/process.h>

#ifdef PSL1GHT
#include <cell/fs/cell_fs_file_api.h>
#include <cell/fs/cell_fs_errno.h>
#endif

#include "app.h"
#include "vfs.h"
#include "http.h"
#include "gfx.h"
#include "theme.h"
#include "pad.h"
#include "audio.h"
#include "font.h"
#include "screen-manager.h"
#include "screens/home.h"
#include "storage.h"
#include "settings.h"
#include "downloads.h"
#include "video-export.h"
#include "ui/console-glyphs.h"
#include "ui/icon-font.h"
#include "ui/stats.h"
#ifndef PSL1GHT
#include "bridge-client.h"
#endif
#include "dbg.h"

#define PROCESS_PRIORITY_DEFAULT 1001
#define PROCESS_STACK_SIZE_64KB  0x10000

SYS_PROCESS_PARAM(PROCESS_PRIORITY_DEFAULT, PROCESS_STACK_SIZE_64KB)

// Keep the PSL1GHT boot trace independent of RTC, VFS, printf, networking and
// graphics.  A launch that returns to XMB can therefore still identify the
// exact subsystem which was entered last on real hardware.
#ifdef PSL1GHT
#define BOOT_LOG "/dev_hdd0/tmp/yo-player-psl1ght-boot.log"

static void bootMark(const char *text, uint64_t length)
{
   int descriptor = -1;
   uint64_t written = 0;
   int flags = CELL_FS_O_WRONLY | CELL_FS_O_CREAT | CELL_FS_O_APPEND;
   if (cellFsOpen(BOOT_LOG, flags, &descriptor, NULL, 0) != CELL_FS_SUCCEEDED) return;
   cellFsWrite(descriptor, text, length, &written);
   cellFsClose(descriptor);
}

#define BOOT_MARK(text) bootMark((text), sizeof(text) - 1)
#else
#define BOOT_MARK(text) ((void)0)
#endif

int main(int argc, char **argv)
{
   (void)argc;
   (void)argv;

   BOOT_MARK("main\n");
   appRegisterExitCallback();
   BOOT_MARK("sysutil callback\n");
   initRtc();
   BOOT_MARK("rtc\n");
   initVfs();                    // file i/o routing (the temp download lands via openFs)
   BOOT_MARK("vfs\n");
   logBuildVersion();
   initTheme();                  // palette from themes.txt: every colour below reads activeTheme
   BOOT_MARK("theme\n");

   int netRc = initNet();
   BOOT_MARK("network\n");
   initModernHttp();   // bind the modern (BearSSL) http transport; all requests + media streams go through it
   BOOT_MARK("http\n");
#ifndef PSL1GHT
   // The loopback debug bridge is a companion to the Sony-SDK development
   // environment.  It is optional and unavailable in a standalone PSL1GHT
   // package, so do not start its background socket thread on retail hardware.
   registerWithBridge("app", "yo-player");
#endif

   if (initGfx(GFX_VSYNC_ON) != 0) { BOOT_MARK("gfx failed\n"); return 1; }
   BOOT_MARK("gfx\n");
   if (initAudio() != 0) { BOOT_MARK("audio failed\n"); return 1; }
   BOOT_MARK("audio\n");
   if (initFont() != 0) { BOOT_MARK("font failed\n"); return 1; }
   BOOT_MARK("font\n");
   if (initIconFont() != 0) logError("[yt] embedded icon font failed to load; icons will be blank\n");
   BOOT_MARK("icon font\n");
   initPad();
   BOOT_MARK("pad\n");

   initStats(5, 5, 14, activeTheme->textSecondary);
   initStorage();          // prefs (last category) + watch history, under /dev_hdd0/tmp/yo-player/
   loadSettings();         // user-editable settings.txt (created with defaults on first launch)
   loadConsoleGlyphs();    // decode the console's own button glyphs for the on-screen hints
   initVideoExport();      // resolves the download staging folder, so it runs before initDownloads
   initDownloads();        // background download queue + its progress overlay
   BOOT_MARK("services\n");

   logInfo("[yt] net rc=%d\n", netRc);
   openHome();       // boot into the home screen (trending categories)
   BOOT_MARK("home\n");

   while (!appExitRequested) {
      appPoll();
      updateVideoExport();   // right after the sysutil pump, which is what delivers its results
      updatePad();
      updateScreen();

      beginGfxFrame();
      clearGfx(activeTheme->appBg);
      drawScreen();
      endGfxFrame();
   }

   changeScreen(NULL);
   shutdownDownloads();   // cancel + join any in-flight download before the http/vfs layers go away
   shutdownVideoExport();
   freeConsoleGlyphs();
   termStats();
   termAudio();
   termFont();
   termGfx();
   shutdownHttp();
   shutdownVfs();
   return 0;
}
