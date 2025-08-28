class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.1.19"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.19/macbat-darwin-arm64.tar.gz"
      sha256 "1b82190808e2b963f1644f7f7665db5e1f91c8fd5de67c8c6662e0fcbc4b5492"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.19/macbat-darwin-amd64.tar.gz"
      sha256 "429ba7e74eb90934ea397a5a1109a74e26291276ff3f53da6d6c8f012bb4c73b"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
