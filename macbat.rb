class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.1.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.2/macbat-darwin-arm64.tar.gz"
      sha256 "8cf366f25f46903917ffc9399dfe2ee9779d60bd30ef993c8aaefb29aa9d78e2"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.2/macbat-darwin-amd64.tar.gz"
      sha256 "8245ca21fb992581cc50378b5818aed168c0f3d5abfada1a774511a76317ec42"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
