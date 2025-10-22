class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.2.9"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.9/macbat-darwin-arm64.tar.gz"
      sha256 "78bc67733b873639272008ed54a88f7701e1d9c7a7c0c6643384fd4b5866207a"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.9/macbat-darwin-amd64.tar.gz"
      sha256 "45bbaef5931b95ff158f3fcd50d0dc6eea427e419570d41a8eb1d49284223179"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
