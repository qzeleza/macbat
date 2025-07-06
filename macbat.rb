class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.1.4"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.4/macbat-darwin-arm64.tar.gz"
      sha256 "555574f326553864cca3640e623c06e7d82cdf86262fc21801bde9aa6dc20a28"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.4/macbat-darwin-amd64.tar.gz"
      sha256 "f937ebfc23fa63257dd465401bcafac8e9491da6ae9444b2d66271f8dd8c0d38"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
