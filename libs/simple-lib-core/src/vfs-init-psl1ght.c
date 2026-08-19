// PSL1GHT VFS bring-up. GameOS already mounts the HDD and FAT32 USB devices;
// the Sony-SDK build's raw-storage NTFS/exFAT drivers are intentionally not
// pulled into this app port.

#include "vfs.h"
#include "vfs-internal.h"

static int initialized;
static void (*mountsChangedCallback)(void);

void setMountsChangedCallback(void (*callback)(void))
{
   mountsChangedCallback = callback;
}

void registerVfsBackend(VfsProbeResult (*probe)(int port),
                        void (*release)(int port), void (*shutdown)(void))
{
   (void)probe; (void)release; (void)shutdown;
}

void initVfs(void)
{
   if (initialized) return;
   ensureMountsLock();
   clearMounts();
   initialized = 1;
   if (mountsChangedCallback) mountsChangedCallback();
}

void shutdownVfs(void)
{
   if (!initialized) return;
   clearMounts();
   initialized = 0;
}
