#!/usr/bin/env python3
"""给 mygo 的 iOS 构建加一个"额外链接参数"的口子。

mygo 把 App 的 Go 代码编成 c-archive，然后用 swiftc 做最终链接
(`cmd/mygo/ios.go` 里 `link := []string{... swiftc ...}`)。c-archive 模式下
cgo 的 LDFLAGS 不参与链接，所以打包进来的 Rust 静态库（eb_* 符号）到不了
最终链接，ld 会报 Undefined symbols。

这个补丁让 mygo 读环境变量 MYGO_IOS_LINK_FLAGS，把它按空格拆开追加到
swiftc 的链接参数里。用法：

    MYGO_IOS_LINK_FLAGS="-force_load /abs/path/libebook_abi.a" mygo build -platform ios

补丁只动 ios.go 里构建 `link` 的那两行，重复执行是安全的（已经打上就不再改）。
"""

import re
import sys

ANCHOR = re.compile(
    r"^(?P<indent>[ \t]*)link = append\(link, frameworks\.\.\.\)[ \t]*$",
    re.MULTILINE,
)

INSERT = (
    "{indent}// 额外的链接输入（App 带来的静态库等），见 MYGO_IOS_LINK_FLAGS。\n"
    "{indent}link = append(link, strings.Fields(os.Getenv(\"MYGO_IOS_LINK_FLAGS\"))...)\n"
)


def main(argv):
    if len(argv) != 1:
        print("usage: patch-mygo-ios-link.py <path/to/cmd/mygo/ios.go>", file=sys.stderr)
        return 2
    path = argv[0]
    with open(path, encoding="utf-8") as handle:
        source = handle.read()

    if "MYGO_IOS_LINK_FLAGS" in source:
        print(f"{path}: 已经打过补丁")
        return 0

    match = ANCHOR.search(source)
    if match is None:
        print(
            f"{path}: 没找到锚点 `link = append(link, frameworks...)`；"
            "mygo 的 ios.go 可能改过，需要手动更新这个补丁",
            file=sys.stderr,
        )
        return 1

    patched = (
        source[: match.end()]
        + "\n"
        + INSERT.format(indent=match.group("indent"))
        + source[match.end():]
    )
    with open(path, "w", encoding="utf-8") as handle:
        handle.write(patched)
    print(f"{path}: 已打补丁（MYGO_IOS_LINK_FLAGS）")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
