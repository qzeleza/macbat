class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.2.5"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.5/macbat-darwin-arm64.tar.gz"
      sha256 "baa95716bb9ba6cc258cd6203cffc881fe542c4429572806e5cb9562321a2547"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.5/macbat-darwin-amd64.tar.gz"
      sha256 "04653ed2317bca52ac6f8a4a9c2f71acd3858599df803d4f6c5d3c393684facc"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
