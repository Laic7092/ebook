#!/usr/bin/env bash
# 构建 Rust 引擎静态库，放进 internal/engine/native/<goos>_<goarch>/。
#
# 用法：
#   scripts/build-engine.sh              # 当前平台（Linux/macOS）
#   scripts/build-engine.sh ios_arm64    # 交叉编译 iOS
#   scripts/build-engine.sh all          # 本机 + 所有能交叉的平台
#
# 环境变量：
#   ENGINE_DIR  ebook-engine 仓库位置（默认 ../ebook-engine）
#   PROFILE     cargo profile（默认 native：opt-level=3 + panic=unwind + 不开 LTO）
set -euo pipefail

cd "$(dirname "$0")/.."
REPO_ROOT=$(pwd)
ENGINE_DIR=${ENGINE_DIR:-$REPO_ROOT/../ebook-engine}
PROFILE=${PROFILE:-native}
STRIP_TOOL=$REPO_ROOT/scripts/strip-bitcode.py

rust_target_of() {
	case "$1" in
	linux_amd64) echo x86_64-unknown-linux-gnu ;;
	linux_arm64) echo aarch64-unknown-linux-gnu ;;
	darwin_amd64) echo x86_64-apple-darwin ;;
	darwin_arm64) echo aarch64-apple-darwin ;;
	ios_arm64) echo aarch64-apple-ios ;;
	ios_sim_arm64) echo aarch64-apple-ios-sim ;;
	android_arm64) echo aarch64-linux-android ;;
	windows_amd64) echo x86_64-pc-windows-gnu ;;
	*) return 1 ;;
	esac
}

host_target() {
	case "$(uname -s)-$(uname -m)" in
	Linux-x86_64) echo linux_amd64 ;;
	Linux-aarch64) echo linux_arm64 ;;
	Darwin-arm64) echo darwin_arm64 ;;
	Darwin-x86_64) echo darwin_amd64 ;;
	*) echo "不认识的宿主平台: $(uname -s)-$(uname -m)" >&2; return 1 ;;
	esac
}

# 去掉静态库成员里的 LLVM bitcode 段：Go 的 cgo 检查会拒绝它们
# （`disallowed LTO section __LLVM,__bitcode in #cgo LDFLAGS`）。
#
# Mach-O（iOS/macOS）用 scripts/strip-bitcode.py 原位改段名——运行器上通常
# 没有 objcopy/segedit，Python 一定有。ELF（Linux）用 objcopy 删段。
strip_bitcode() {
	local archive=$1
	python3 "$STRIP_TOOL" "$archive" || true
	for tool in objcopy llvm-objcopy "xcrun llvm-objcopy"; do
		# shellcheck disable=SC2086
		$tool --remove-section=.llvmbc --remove-section=.llvmcmd "$archive" 2>/dev/null && break
	done
	if ! python3 "$STRIP_TOOL" --check "$archive"; then
		echo "    警告：$archive 里还有 bitcode 段，Go 链接会失败" >&2
		return 1
	fi
}

build_one() {
	local go_target=$1
	local rust_target
	rust_target=$(rust_target_of "$go_target") || {
		echo "跳过 $go_target（没有对应的 Rust target）" >&2
		return 0
	}

	echo "==> $go_target ($rust_target)"
	# 目标平台要在引擎仓库里装：那里有 rust-toolchain.toml，rustup 会按
	# 固定的工具链来解析，装到别处等于没装。
	(
		cd "$ENGINE_DIR"
		rustup target list --installed | grep -qx "$rust_target" ||
			rustup target add "$rust_target"
		cargo build --profile "$PROFILE" --target "$rust_target" -p ebook-abi --features parallel
	)

	local built="$ENGINE_DIR/target/$rust_target/$PROFILE/libebook_abi.a"
	[ -f "$built" ] || {
		echo "没有找到 $built" >&2
		return 1
	}

	local dest="$REPO_ROOT/internal/engine/native/$go_target"
	mkdir -p "$dest"
	cp "$built" "$dest/libebook_abi.a"
	strip_bitcode "$dest/libebook_abi.a"
	ls -lh "$dest/libebook_abi.a" | awk '{print "    " $5 "  " $9}'
}

if [ $# -eq 0 ]; then
	build_one "$(host_target)"
	exit 0
fi

for arg in "$@"; do
	if [ "$arg" = "all" ]; then
		build_one "$(host_target)" || true
		build_one ios_arm64 || true
		build_one ios_sim_arm64 || true
	else
		build_one "$arg"
	fi
done
