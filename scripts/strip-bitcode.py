#!/usr/bin/env python3
"""把静态库成员里的 LLVM bitcode 段改个名字，让 Go 的链接器不再拒绝它。

为什么需要：Go 在链接 cgo 目标时会检查 #cgo LDFLAGS 里的归档，发现
`.llvmbc`（ELF）或 `__LLVM,__bitcode`（Mach-O）就直接报错：

    disallowed LTO section __LLVM,__bitcode in #cgo LDFLAGS: libebook_abi.a

这些段来自 Rust 预编译的 std 对象，删掉它们不影响代码（它们只是让 LTO
能跨 crate 内联的附加数据），但 Go 不认识、也不想认识。ELF 上有 objcopy
可以删段；macOS 的运行器上通常没有 objcopy/segedit 能这么干，所以这里
直接按 Mach-O 的结构把段名改掉（同为 16 字节，原位覆盖）。

用法：
    strip-bitcode.py <archive-or-object>...
退出码 0 表示“处理过或不需要处理”，1 表示有残余的 bitcode 段。
"""

import struct
import sys

LC_SEGMENT = 0x1
LC_SEGMENT_64 = 0x19
MH_MAGIC_64 = 0xFEEDFACF
MH_MAGIC = 0xFEEDFACE

# 段名/section 名都是 16 字节定长，改名后长度不变。
#
# Go 的检查（cmd/go/internal/work/security.go）认的是 `sec.Seg == "__LLVM"`，
# 所以只改 section 名没用，**段名**也得改；section 自己的 segname 字段同样要改
# （debug/macho 的 Section.Seg 来自那里）。
SEGMENT_RENAMES = {
    b"__LLVM": b"__LLVX",
    b"__GNU_LTO": b"__GNULTO",
}
RENAMES = {
    b"__bitcode": b"__bitcod_",
    b"__cmdline": b"__cmdlin_",
}


def patch_macho64(data: bytearray, base: int, size: int) -> int:
    """改写 base..base+size 之间一个 Mach-O 文件里的 bitcode 段名。

    返回改写的段数。只处理 64 位小端（Apple Silicon / Intel macOS）。
    """
    if size < 32:
        return 0
    magic = struct.unpack_from("<I", data, base)[0]
    if magic != MH_MAGIC_64:
        return 0

    ncmds = struct.unpack_from("<I", data, base + 16)[0]
    offset = base + 32
    patched = 0
    for _ in range(ncmds):
        if offset + 8 > base + size:
            break
        cmd, cmdsize = struct.unpack_from("<II", data, offset)
        if cmdsize < 8:
            break
        if cmd == LC_SEGMENT_64 and offset + 72 <= base + size:
            segname = bytes(data[offset + 8:offset + 24]).rstrip(b"\x00")
            if segname in SEGMENT_RENAMES:
                data[offset + 8:offset + 24] = SEGMENT_RENAMES[segname].ljust(16, b"\x00")
                patched += 1
            nsects = struct.unpack_from("<I", data, offset + 64)[0]
            sect = offset + 72
            for _ in range(nsects):
                if sect + 80 > base + size:
                    break
                name = bytes(data[sect:sect + 16]).rstrip(b"\x00")
                replacement = RENAMES.get(name)
                if replacement is not None:
                    data[sect:sect + 16] = replacement.ljust(16, b"\x00")
                    patched += 1
                sec_seg = bytes(data[sect + 16:sect + 32]).rstrip(b"\x00")
                if sec_seg in SEGMENT_RENAMES:
                    data[sect + 16:sect + 32] = SEGMENT_RENAMES[sec_seg].ljust(16, b"\x00")
                    patched += 1
                sect += 80
        offset += cmdsize
    return patched


def archive_members(data: bytearray):
    """遍历 ar 归档，产出 (数据起始偏移, 长度)。"""
    pos = 8  # "!<arch>\n"
    while pos + 60 <= len(data):
        header = data[pos:pos + 60]
        if header[58:60] != b"`\n":
            return
        name = bytes(header[0:16]).rstrip(b" ")
        size = int(header[48:58].decode("ascii", "replace").strip() or 0)
        start = pos + 60
        # BSD ar 把长名字直接放在成员数据前面：`#1/<len>`
        if name.startswith(b"#1/"):
            try:
                name_len = int(name[3:])
            except ValueError:
                name_len = 0
            start += name_len
            size -= name_len
        yield start, size
        pos = start + size + (size & 1)


def has_bitcode(data: bytes) -> bool:
    """Go 会拒绝的特征：ELF 的 .llvmbc/.llvmcmd，或者 Mach-O 的 __LLVM 段。"""
    return (
        b"__bitcode" in data
        or b".llvmbc" in data
        or b".llvmcmd" in data
        or b"__LLVM\x00" in data
        or b"__GNU_LTO\x00" in data
    )


def main(argv):
    check_only = False
    if argv and argv[0] == "--check":
        check_only = True
        argv = argv[1:]
    status = 0
    for path in argv:
        try:
            with open(path, "rb") as handle:
                data = bytearray(handle.read())
        except OSError as error:
            print(f"strip-bitcode: 读不了 {path}: {error}", file=sys.stderr)
            status = 1
            continue

        if check_only:
            if has_bitcode(bytes(data)):
                print(f"strip-bitcode: {path}: 还有 bitcode 段", file=sys.stderr)
                status = 1
            continue

        if data[:8] == b"!<arch>\n":
            patched = sum(patch_macho64(data, start, size) for start, size in archive_members(data))
        else:
            patched = patch_macho64(data, 0, len(data))

        if patched:
            with open(path, "wb") as handle:
                handle.write(data)
            print(f"strip-bitcode: {path}: 改了 {patched} 个段名")
        elif has_bitcode(bytes(data)):
            print(
                f"strip-bitcode: {path}: 仍然有 bitcode（可能是 ELF，请用 objcopy）",
                file=sys.stderr,
            )
            status = 1
    return status


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
