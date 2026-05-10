#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# Linux only. Installs runtime or development system dependencies for
# Ebitengine's GLFW backend. Use 'dev' for build headers/toolchain,
# otherwise runtime libraries are installed. Detects apt, dnf, yum,
# pacman, zypper, apk.

set -euo pipefail

log() { printf '\033[1;34m[install-deps]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[install-deps]\033[0m %s\n' "$*" >&2; }
die() {
    printf '\033[1;31m[install-deps]\033[0m %s\n' "$*" >&2
    exit 1
}

mode=runtime

case "${1:-}" in
    -h | --help)
        sed -n '2,/^$/p' "$0" | sed 's/^# \{0,1\}//'
        exit 0
        ;;
    "") ;;
    runtime) ;;
    dev) mode=dev ;;
    *) die "unknown flag: $1" ;;
esac

need_root() {
    if [ "$(id -u)" -eq 0 ]; then return; fi
    die "must run as root"
}

detect_os() {
    case "$(uname -s)" in
        Linux) echo linux ;;
        *) die "unsupported OS: $(uname -s)" ;;
    esac
}

detect_pm() {
    if command -v apt-get >/dev/null 2>&1; then
        echo apt
    elif command -v dnf >/dev/null 2>&1; then
        echo dnf
    elif command -v yum >/dev/null 2>&1; then
        echo yum
    elif command -v pacman >/dev/null 2>&1; then
        echo pacman
    elif command -v zypper >/dev/null 2>&1; then
        echo zypper
    elif command -v apk >/dev/null 2>&1; then
        echo apk
    else
        die "no supported package manager found"
    fi
}

install_linux_deps() {
    local mode=$1
    local pm
    pm=$(detect_pm)
    log "package manager: $pm"
    log "install mode: $mode"
    case "$pm" in
        apt)
            export DEBIAN_FRONTEND=noninteractive
            apt-get update -qq
            if [ "$mode" = "dev" ]; then
                apt-get install -y --no-install-recommends \
                    build-essential pkg-config \
                    libx11-dev libxcursor-dev libxinerama-dev libxi-dev \
                    libxrandr-dev libxxf86vm-dev libxkbcommon-dev \
                    libwayland-dev libgl1-mesa-dev libasound2-dev
            else
                apt-get install -y --no-install-recommends \
                    libx11-6 libxcursor1 libxinerama1 libxi6 \
                    libxrandr2 libxxf86vm1 libxkbcommon0 \
                    libwayland-client0 libgl1 libasound2
            fi
            ;;
        dnf | yum)
            if [ "$mode" = "dev" ]; then
                "$pm" install -y \
                    gcc gcc-c++ make pkgconf-pkg-config \
                    libX11-devel libXcursor-devel libXinerama-devel libXi-devel \
                    libXrandr-devel libXxf86vm-devel libxkbcommon-devel \
                    wayland-devel mesa-libGL-devel alsa-lib-devel
            else
                "$pm" install -y \
                    libX11 libXcursor libXinerama libXi \
                    libXrandr libXxf86vm libxkbcommon \
                    wayland mesa-libGL alsa-lib
            fi
            ;;
        pacman)
            if [ "$mode" = "dev" ]; then
                pacman -Syu --noconfirm --needed \
                    base-devel pkgconf \
                    libx11 libxcursor libxinerama libxi libxrandr libxxf86vm \
                    libxkbcommon wayland mesa alsa-lib
            else
                pacman -Syu --noconfirm --needed \
                    libx11 libxcursor libxinerama libxi libxrandr libxxf86vm \
                    libxkbcommon wayland mesa alsa-lib
            fi
            ;;
        zypper)
            if [ "$mode" = "dev" ]; then
                zypper --non-interactive install -y \
                    gcc-c++ make pkg-config \
                    libX11-devel libXcursor-devel libXinerama-devel libXi-devel \
                    libXrandr-devel libXxf86vm-devel libxkbcommon-devel \
                    wayland-devel Mesa-libGL-devel alsa-devel
            else
                zypper --non-interactive install -y \
                    libX11-6 libXcursor1 libXinerama1 libXi6 \
                    libXrandr2 libXxf86vm1 libxkbcommon0 \
                    libwayland-client0 libOpenGL1 libasound2
            fi
            ;;
        apk)
            if [ "$mode" = "dev" ]; then
                apk add \
                    build-base pkgconf \
                    libx11-dev libxcursor-dev libxinerama-dev libxi-dev \
                    libxrandr-dev libxxf86vm-dev libxkbcommon-dev \
                    wayland-dev mesa-dev alsa-lib-dev
            else
                apk add \
                    libx11 libxcursor libxinerama libxi libxrandr libxxf86vm \
                    libxkbcommon wayland-libs-client mesa-gl alsa-lib
            fi
            ;;
    esac
    log "system packages installed"
}

main() {
    local os
    os=$(detect_os)
    log "OS: $os"
    case "$os" in
        linux)
            need_root "$@"
            install_linux_deps "$mode"
            ;;
    esac
}

main "$@"
