class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.1.7"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.7/macbat-darwin-arm64.tar.gz"
      sha256 "0146e23ad293d5554c90a9f2ecb29cc4adf3fc59c451934d62e4023ee5aa55ed"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.7/macbat-darwin-amd64.tar.gz"
      sha256 "19d6ae769673f4b9821a4f7b0c758fa0c0151acbb2a769f2b4b7f0ccf081285b"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
