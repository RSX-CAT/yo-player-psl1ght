#!/usr/bin/env python3
"""Set the SDK field in a PSL1GHT executable's PRX process parameters.

PSL1GHT's lv2-sprx.o currently emits zero for this field.  GameOS uses the
PRX metadata independently of .sys_proc_param when it resolves dependencies
between dynamically loaded system modules (notably libfontFT -> libfont).
"""

from __future__ import annotations

import argparse
import struct
from pathlib import Path


SECTION_NAME = b".sys_proc_prx_param"
PRX_PARAM_SIZE = 0x28
PRX_PARAM_MAGIC = 0x1B434CEC
PRX_PARAM_VERSION = 2
SDK_FIELD_OFFSET = 0x24


def section_location(data: bytes, wanted: bytes) -> tuple[int, int]:
    if data[:6] != b"\x7fELF\x02\x02":
        raise ValueError("expected a big-endian ELF64 executable")

    section_offset = struct.unpack_from(">Q", data, 0x28)[0]
    section_entry_size, section_count, names_index = struct.unpack_from(
        ">HHH", data, 0x3A
    )
    if not section_offset or not section_entry_size or names_index >= section_count:
        raise ValueError("ELF section table is missing or invalid")

    headers = [
        struct.unpack_from(">IIQQQQIIQQ", data, section_offset + i * section_entry_size)
        for i in range(section_count)
    ]
    names_header = headers[names_index]
    names = data[names_header[4] : names_header[4] + names_header[5]]

    for header in headers:
        name_offset = header[0]
        if name_offset >= len(names):
            continue
        name = names[name_offset:].split(b"\0", 1)[0]
        if name == wanted:
            return header[4], header[5]
    raise ValueError(f"ELF section {wanted.decode()} was not found")


def set_sdk_version(path: Path, sdk_version: int) -> None:
    data = bytearray(path.read_bytes())
    offset, size = section_location(data, SECTION_NAME)
    if size != PRX_PARAM_SIZE or offset + size > len(data):
        raise ValueError(f"unexpected PRX parameter size 0x{size:x}")

    param_size, magic, version = struct.unpack_from(">III", data, offset)
    if (param_size, magic, version) != (
        PRX_PARAM_SIZE,
        PRX_PARAM_MAGIC,
        PRX_PARAM_VERSION,
    ):
        raise ValueError(
            "unexpected PRX parameter header: "
            f"size=0x{param_size:x}, magic=0x{magic:08x}, version={version}"
        )

    old_version = struct.unpack_from(">I", data, offset + SDK_FIELD_OFFSET)[0]
    if old_version not in (0, sdk_version):
        raise ValueError(
            f"refusing to replace existing PRX SDK version 0x{old_version:08x}"
        )

    struct.pack_into(">I", data, offset + SDK_FIELD_OFFSET, sdk_version)
    path.write_bytes(data)
    print(
        f"PRX SDK version: 0x{old_version:08x} -> 0x{sdk_version:08x} "
        f"({path})"
    )


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("elf", type=Path)
    parser.add_argument("--sdk", type=lambda value: int(value, 0), required=True)
    args = parser.parse_args()
    if not 0 <= args.sdk <= 0xFFFFFFFF:
        parser.error("--sdk must fit in 32 bits")
    set_sdk_version(args.elf, args.sdk)


if __name__ == "__main__":
    main()
