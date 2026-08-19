#pragma once

// The compatibility wrappers normalize the two idempotent operations before
// cellfs.c sees them, so these values are only retained for its comparisons.
#define CELL_FS_EEXIST 0x80010014u
#define CELL_FS_ENOENT 0x80010006u
