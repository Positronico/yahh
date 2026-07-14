#!/usr/bin/env bash
# Generates Formula/yahh.rb for a released version, filling the sha256 of
# every artifact from the release's checksums.txt.
#
# Usage:
#   packaging/update-formula.sh vX.Y.Z [checksums.txt] > <tap>/Formula/yahh.rb
#
# Without the second argument the checksums are fetched from the GitHub
# release; pass a local file (e.g. dist/checksums.txt) to test offline.
set -euo pipefail

TAG="${1:?usage: update-formula.sh vX.Y.Z [checksums.txt]}"
VERSION="${TAG#v}"

if [[ $# -ge 2 ]]; then
  CHECKSUMS=$(cat "$2")
else
  CHECKSUMS=$(curl -fsSL "https://github.com/Positronico/yahh/releases/download/${TAG}/checksums.txt")
fi

sha() {
  local file="yahh_${VERSION}_$1.tar.gz" hash
  hash=$(awk -v f="yahh_${VERSION}_$1.tar.gz" '$2 == f {print $1}' <<<"$CHECKSUMS")
  if [[ -z "$hash" ]]; then
    echo "error: $file not found in checksums" >&2
    exit 1
  fi
  echo "$hash"
}

cat <<EOF
# Homebrew formula for yahh. Lives in github.com/Positronico/homebrew-tap
# as Formula/yahh.rb; users install with:
#   brew install Positronico/tap/yahh
#
# Maintained manually: regenerate for each release with
#   packaging/update-formula.sh v${VERSION} > Formula/yahh.rb
# from the Positronico/yahh repo. See its PUBLISHING.md.
class Yahh < Formula
  desc "Per-project shell history realms for zsh and bash"
  homepage "https://github.com/Positronico/yahh"
  version "${VERSION}"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Positronico/yahh/releases/download/v#{version}/yahh_#{version}_darwin_arm64.tar.gz"
      sha256 "$(sha darwin_arm64)"
    end
    on_intel do
      url "https://github.com/Positronico/yahh/releases/download/v#{version}/yahh_#{version}_darwin_amd64.tar.gz"
      sha256 "$(sha darwin_amd64)"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Positronico/yahh/releases/download/v#{version}/yahh_#{version}_linux_arm64.tar.gz"
      sha256 "$(sha linux_arm64)"
    end
    on_intel do
      url "https://github.com/Positronico/yahh/releases/download/v#{version}/yahh_#{version}_linux_amd64.tar.gz"
      sha256 "$(sha linux_amd64)"
    end
  end

  def install
    bin.install "yahh"
    generate_completions_from_executable(bin/"yahh", "completion")
  end

  def caveats
    <<~EOS
      To activate yahh, add to your shell rc file (or run \`yahh install\`):
        eval "\$(yahh init zsh)"    # ~/.zshrc
        eval "\$(yahh init bash)"   # ~/.bashrc
    EOS
  end

  test do
    system "#{bin}/yahh", "version"
  end
end
EOF
