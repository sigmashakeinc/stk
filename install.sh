#!/bin/sh
# Install stk (Sigma Token Killer) on Linux or macOS.
#
# Usage: install.sh [--version <tag>] [--from-file <archive> [--checksums <file>]]
#
# Environment:
#   INSTALL_DIR        Install directory. Default: $HOME/.local/bin
#   VERSION            Release tag to install (example: v0.1.0). Default: latest.
#   STK_FROM_FILE      Same as --from-file.
#   STK_CHECKSUMS_FILE Same as --checksums.
set -eu

REPO="sigmashakeinc/stk"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-}"
FROM_FILE="${STK_FROM_FILE:-}"
CHECKSUMS_FILE="${STK_CHECKSUMS_FILE:-}"

die() {
	printf 'install.sh: %s\n' "$*" >&2
	exit 1
}

while [ $# -gt 0 ]; do
	case "$1" in
	--version)
		[ $# -ge 2 ] || die "--version needs a value"
		VERSION="$2"
		shift 2
		;;
	--from-file)
		[ $# -ge 2 ] || die "--from-file needs a path"
		FROM_FILE="$2"
		shift 2
		;;
	--checksums)
		[ $# -ge 2 ] || die "--checksums needs a path"
		CHECKSUMS_FILE="$2"
		shift 2
		;;
	-h | --help)
		printf 'Usage: install.sh [--version <tag>] [--from-file <archive> [--checksums <file>]]\n'
		printf 'Env: INSTALL_DIR (default $HOME/.local/bin), VERSION, STK_FROM_FILE, STK_CHECKSUMS_FILE\n'
		exit 0
		;;
	*) die "unknown option: $1" ;;
	esac
done

raw_os="$(uname -s | tr '[:upper:]' '[:lower:]')"
raw_arch="$(uname -m)"
case "$raw_arch" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) arch="$raw_arch" ;;
esac
case "$raw_os/$arch" in
linux/amd64) os=linux ;;
darwin/arm64) os=darwin ;;
*)
	printf 'stk: unsupported platform %s/%s; supported: linux x64, macOS arm64, Windows x64\n' "$raw_os" "$arch" >&2
	exit 1
	;;
esac

if command -v sha256sum >/dev/null 2>&1; then
	hash_cmd="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
	hash_cmd="shasum -a 256"
else
	die "need sha256sum or shasum to verify the download"
fi

fetch() {
	# fetch <url> <output>
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$1" -o "$2"
	elif command -v wget >/dev/null 2>&1; then
		wget -q "$1" -O "$2"
	else
		die "need curl or wget"
	fi
}

tmp="$(mktemp -d 2>/dev/null || mktemp -d -t stk)"

# Remove only the files this script creates, then the empty directories.
cleanup() {
	if [ -n "$archive_name_dl" ]; then
		rm -f "$tmp/$archive_name_dl" "$tmp/checksums.txt"
	fi
	rm -f "$tmp/x/stk"
	rmdir "$tmp/x" 2>/dev/null || true
	rmdir "$tmp" 2>/dev/null || true
}
archive_name_dl=""
trap cleanup EXIT
trap 'exit 1' INT TERM

if [ -n "$FROM_FILE" ]; then
	[ -f "$FROM_FILE" ] || die "archive not found: $FROM_FILE"
	archive="$FROM_FILE"
	archive_name="$(basename "$FROM_FILE")"
	if [ -n "$CHECKSUMS_FILE" ]; then
		[ -f "$CHECKSUMS_FILE" ] || die "checksums file not found: $CHECKSUMS_FILE"
		sums="$CHECKSUMS_FILE"
	else
		sums=""
		printf 'install.sh: warning: no checksums file given. Skipping verification of the local archive.\n' >&2
	fi
else
	if [ -z "$VERSION" ]; then
		# The /releases/latest URL redirects to /releases/tag/<tag>.
		if command -v curl >/dev/null 2>&1; then
			url="$(curl -fsSIL -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest")" ||
				die "cannot resolve the latest release"
		else
			url="$(wget --max-redirect=0 -S -O /dev/null "https://github.com/$REPO/releases/latest" 2>&1 |
				sed -n 's/^ *[Ll]ocation: *//p' | tr -d '\r' | head -n 1)" ||
				die "cannot resolve the latest release"
		fi
		VERSION="${url##*/}"
		[ -n "$VERSION" ] || die "cannot resolve the latest release"
	fi
	archive_name="stk_${VERSION}_${os}_${arch}.tar.gz"
	archive_name_dl="$archive_name"
	base="https://github.com/$REPO/releases/download/$VERSION"
	archive="$tmp/$archive_name"
	sums="$tmp/checksums.txt"
	printf 'Downloading %s\n' "$archive_name"
	fetch "$base/$archive_name" "$archive" || die "download failed: $base/$archive_name"
	fetch "$base/checksums.txt" "$sums" || die "download failed: $base/checksums.txt"
fi

if [ -n "$sums" ]; then
	expected="$(awk -v n="$archive_name" '$2 == n || $2 == "*" n { print $1; exit }' "$sums")"
	[ -n "$expected" ] || die "no checksum entry for $archive_name"
	actual="$($hash_cmd "$archive" | awk '{ print $1 }')"
	if [ "$expected" != "$actual" ]; then
		die "checksum mismatch for $archive_name (expected $expected, got $actual)"
	fi
	printf 'Checksum OK\n'
fi

mkdir "$tmp/x"
tar -xzf "$archive" -C "$tmp/x" || die "cannot extract $archive_name"
[ -f "$tmp/x/stk" ] || die "archive has no stk binary"

mkdir -p "$INSTALL_DIR"
cp "$tmp/x/stk" "$INSTALL_DIR/stk.new"
chmod 755 "$INSTALL_DIR/stk.new"
mv -f "$INSTALL_DIR/stk.new" "$INSTALL_DIR/stk"
printf 'Installed stk to %s/stk\n' "$INSTALL_DIR"

case ":$PATH:" in
*":$INSTALL_DIR:"*) ;;
*)
	printf '\n%s is not on your PATH. Add this line to your shell profile:\n' "$INSTALL_DIR"
	printf '  export PATH="%s:$PATH"\n' "$INSTALL_DIR"
	;;
esac
