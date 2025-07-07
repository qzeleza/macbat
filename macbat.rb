class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.1.8"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.8/macbat-darwin-arm64.tar.gz"
      sha256 "3e961adb810981be9f9ead669e8cbe1de75956acaa9c923c973a10b8e6d01271"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.8/macbat-darwin-amd64.tar.gz"
      sha256 "b0169b6af771966c706ceb5d38215d08ff131932b52ab125ee2769ed6e7cdbe0"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
