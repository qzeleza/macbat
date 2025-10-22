class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.2.10"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.10/macbat-darwin-arm64.tar.gz"
      sha256 "192e3cda2ebb6e1a321c908b039af8118c513007653c698c61868ee48766ef57"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.10/macbat-darwin-amd64.tar.gz"
      sha256 "67d47846d53f5ef3f81629d71db8aa65463c5fd8bfdcd63610b43ebb33bba620"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
