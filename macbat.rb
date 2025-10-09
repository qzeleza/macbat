class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.1.23"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.23/macbat-darwin-arm64.tar.gz"
      sha256 "3fb8efe86ab4a6e96b82920714ee15634c8e1adb52cdb2f18810a73075f70ce4"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.23/macbat-darwin-amd64.tar.gz"
      sha256 "89152e9f437c02e3dd0eb1e4ca34a5f4d733bdaa25b0b977260b7b038a6c4164"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
