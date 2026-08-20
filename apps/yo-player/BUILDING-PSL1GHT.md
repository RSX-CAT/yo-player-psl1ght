# Yo! Player: PSL1GHT build

This is the open-toolchain build of Yo! Player. It preserves the existing Sony
SDK project and selects PSL1GHT-specific implementations from the Makefiles.
The package uses title ID `YOPSL0001`, so it installs beside the official-SDK
release (`YOPLAYER1`) instead of replacing a known-good installation. Both
builds intentionally share settings, history, and downloads below
`/dev_hdd0/tmp/yo-player/`.

## Supported build hosts

PS3DEV currently supports native Linux and macOS. Its documented Windows route
is Ubuntu under WSL. If a local WSL/VM is undesirable, the included GitHub
Actions workflow builds the package on a Linux runner; no local VM is required.

The build is pinned in CI to the official PS3DEV pre-release
`nightly-2026-07-26` and verifies the Linux archive's published SHA-256 digest.
A local installation needs the complete prebuilt PS3DEV archive, not only a PPU
compiler: PSL1GHT, package tools, portlibs, FAAD2, and `cgcomp` are all used.

Set the standard SDK environment:

```sh
export PS3DEV=/usr/local/ps3dev
export PSL1GHT="$PS3DEV"
export PATH="$PS3DEV/bin:$PS3DEV/ppu/bin:$PS3DEV/spu/bin:$PATH"
```

Sanity-check the required pieces:

```sh
ppu-gcc --version
test -x "$PS3DEV/bin/cgcomp"
test -f "$PS3DEV/portlibs/ppu/include/neaacdec.h"
test -f "$PS3DEV/portlibs/ppu/lib/libfaad.a"
```

## Build

From the repository root:

```sh
make -C apps/yo-player -j"$(nproc)"
make -C apps/yo-player pkg
```

The first command produces `yo-player-psl1ght.elf`, a CEX SELF, and a fake
SELF. The second also produces `yo-player-psl1ght.pkg` and the finalized NPDRM
package, `yo-player-psl1ght.gnpdrm.pkg`. `make -C apps/yo-player clean`
removes only PSL1GHT outputs.

The shaders are RSX assembly compiled with PSL1GHT's `cgcomp -a`; Nvidia Cg and
Sony's Cg compiler are not required.

## Install on Evilnat CFW

Copy `apps/yo-player/yo-player-psl1ght.gnpdrm.pkg` to a FAT32 USB drive's
`/packages` directory or to `/dev_hdd0/packages`, then install it through
Package Manager. The distinct title ID makes rollback simple: uninstall the
PSL1GHT entry and the official release remains installed.

For quick development, `make -C apps/yo-player run` uses `ps3load` and sends the
generated SELF to a listening console.

## Port behavior

| Subsystem | PSL1GHT implementation |
| --- | --- |
| UI/video output | native libRSX; fixed attribute slots and assembly shaders |
| H.264 | GameOS VDEC through PSL1GHT, three SPUs, YUV420 output |
| AAC | FAAD2 from ps3libraries, stereo float output |
| HTTPS | bundled BearSSL and PSL1GHT BSD sockets |
| Images/fonts/input/audio | PSL1GHT PNG/JPEG, font, pad, OSK, and audio APIs |
| Files | GameOS HDD and kernel-mounted FAT32 devices through `sysFs` |
| Completed downloads | retained in `/dev_hdd0/tmp/yo-player/downloads` |

Two Sony-only features do not have public PSL1GHT equivalents:

- The Video Export utility cannot insert a completed file into the XMB media
  database. The port keeps the file in the app-owned downloads directory.
- The original repository's custom raw NTFS/exFAT backends are not included in
  this app target. GameOS HDD and FAT32 USB paths remain available.

## Hardware validation checklist

After installing, keep the official release for comparison and test in this
order:

1. Boot, XMB exit callback, pad navigation, OSK search, and HTTPS thumbnails.
2. One 720p H.264/AAC video: start, pause, seek forward/back, and exit playback.
3. A longer video while showing the statistics overlay; watch for VDEC backlog,
   audio underruns, and flip waits.
4. Download a video and verify that it remains in the downloads directory.
5. Repeat at the console's 480p/576p/720p/1080p output modes that you use.

Runtime diagnostics are appended to `/dev_hdd0/tmp/dbg.txt`. The open-toolchain
build also writes dependency-free startup checkpoints to
`/dev_hdd0/tmp/yo-player-psl1ght-boot.log`; the final line identifies the last
subsystem reached if GameOS returns to XMB during startup.

## Licensing note

The source repository is Apache-2.0, BearSSL is MIT, and FAAD2 2.7 is
GPL-2.0-or-later. A distributed binary linked with FAAD2 should therefore be
treated as a GPLv3 combined work: preserve notices, provide the corresponding
source and build scripts, and distribute under GPLv3-compatible terms. This is
a practical compliance note, not legal advice.
