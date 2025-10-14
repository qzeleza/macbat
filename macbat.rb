class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.2.6"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.6/macbat-darwin-arm64.tar.gz"
      sha256 "9c63b6cd0fce157905c86ac13e0e78ed575cd35e61babf0d9ae5de844a5eab4a"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.6/macbat-darwin-amd64.tar.gz"
      sha256 "066bde87ab6f783b2d39d5abfc0b0aae444b2258c4ce61e40594c5ce327d29a8"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
