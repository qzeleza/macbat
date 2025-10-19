class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.2.8"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.8/macbat-darwin-arm64.tar.gz"
      sha256 "461e724b260ea0585f615e07898b3b37b28da3e825c3f595bcc2674a15c5b1f5"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.2.8/macbat-darwin-amd64.tar.gz"
      sha256 "4528f34015304862b9811eb22025130015f570829215e5ac92cb81de3cf22453"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
