#pragma once

// Small source-compatibility layer for code shared with the official-SDK build.
// Platform-specific subsystems (VDEC/AAC/video export) use native PSL1GHT
// implementations; this file only covers stable ABI-equivalent names.

#include <ppu-types.h>
#include <net/net.h>

#ifndef CELL_OK
#define CELL_OK 0
#endif

#ifndef socketclose
#define socketclose netClose
#endif

#ifndef BUILD_NAME
#define BUILD_NAME "yo-player-psl1ght"
#endif

#ifndef BUILD_STAMP
#define BUILD_STAMP "development"
#endif
