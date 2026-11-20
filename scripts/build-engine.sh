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
#
# 为什么要 `objcopy --remove-section=.llvmbc`：Rust 的 std 预编译对象里带
# LLVM bitcode 段，Go 的链接器见到 `.llvmbc` 会直接拒绝（disallowed LTO
# section）。去掉这两个段不影响代码，只是让 Go 的检查满意。
set -euo pipefail

cd "$(dirname "$0")/.."
REPO_ROOT=$(pwd)
ENGINE_DIR=${ENGINE_DIR:-$REPO_ROOT/../ebook-engine}
PROFILE=${PROFILE:-native}

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

build_one() {
	local go_target=$1
	local rust_target
	rust_target=$(rust_target_of "$go_target") || {
		echo "跳过 $go_target（没有对应的 Rust target）" >&2
		return 0
	}

	echo "==> $go_target ($rust_target)"
	if ! rustup target list --installed | grep -qx "$rust_target"; then
		echo "    rustup target add $rust_target"
		rustup target add "$rust_target" >/dev/null
	fi

	(cd "$ENGINE_DIR" && cargo build --profile "$PROFILE" --target "$rust_target" -p ebook-abi --features parallel)

	local built="$ENGINE_DIR/target/$rust_target/$PROFILE/libebook_abi.a"
	[ -f "$built" ] || {
		echo "没有找到 $built" >&2
		return 1
	}

	local dest="$REPO_ROOT/internal/engine/native/$go_target"
	mkdir -p "$dest"
	cp "$built" "$dest/libebook_abi.a"
	if command -v objcopy >/dev/null 2>&1; then
		objcopy --remove-section=.llvmbc --remove-section=.llvmcmd "$dest/libebook_abi.a" 2>/dev/null || true
	elif command -v llvm-objcopy >/dev/null 2>&1; then
		llvm-objcopy --remove-section=.llvmbc --remove-section=.llvmcmd "$dest/libebook_abi.a" 2>/dev/null || true
	else
		echo "    警告：没有 objcopy，Go 链接可能因为 .llvmbc 段失败" >&2
	fi
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
