class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.2.7"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.7/macbat-darwin-arm64.tar.gz"
      sha256 "0eb0c038badb221e027be70a28fed22321d1dd190fa5b432df01bbc1d97e47b4"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.7/macbat-darwin-amd64.tar.gz"
      sha256 "4b7c8a763cf696f9adbf6e770ab4859ddff007df3dcd1c20bf343c1ae218cfa8"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
