class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.2.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.3/macbat-darwin-arm64.tar.gz"
      sha256 "153badeea56c7d4af5fef0341d61afa04d02f6284af0e91fa10535143c37464b"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.3/macbat-darwin-amd64.tar.gz"
      sha256 "a195eaf6c45291781b54d338d0112799fee3008342d143b29562df097c9a3ea4"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
