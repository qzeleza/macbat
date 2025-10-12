class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.2.4"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.4/macbat-darwin-arm64.tar.gz"
      sha256 "849ce7478c590e9fd53452e81da42112043f2344726b93ff7ddeface99464ccd"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.4/macbat-darwin-amd64.tar.gz"
      sha256 "35371969a55285bbbc0f0bf916407719940e6768d4730cbb9065c777b64d4efd"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
