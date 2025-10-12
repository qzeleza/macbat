class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.2.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.1/macbat-darwin-arm64.tar.gz"
      sha256 "c580146ec04bf729d440d9d3488c835961f02a2f5197cee4ca3991681d04a4cf"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.1/macbat-darwin-amd64.tar.gz"
      sha256 "296414205575c707ed2820f89268bfcc19e30b0fe39a120cc60673dfc6e47237"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
