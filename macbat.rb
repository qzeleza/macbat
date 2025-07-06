class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.1.5"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.5/macbat-darwin-arm64.tar.gz"
      sha256 "e380fb45505f593f53c7d5b70111311eee1498f59d224e5574b431931fb1d15d"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.5/macbat-darwin-amd64.tar.gz"
      sha256 "36d48e550c252d2df0d93f6ff6c52ed49b7f4fb4bcdb67f96f06436bcacf26fa"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
