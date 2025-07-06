class Macbat < Formula
  desc "Утилита мониторинга аккумулятора (binary)"
  homepage "https://github.com/qzeleza/macbat"
  version "v2.1.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.3/macbat-darwin-arm64.tar.gz"
      sha256 "af9597dc07d4f97c22499ca813965b08f7fbec2dc936d9de843619c120322ea1"
    else
      url "https://github.com/qzeleza/macbat/releases/download/v2.1.3/macbat-darwin-amd64.tar.gz"
      sha256 "1c054503dad649c2e0e8cbb240394ccdd4a4d958dd9b94a1fb118ee0e409d507"
    end
  end

  def install
    bin.install "macbat"
  end

  test do
    system "#{bin}/macbat", "--version"
  end
end
