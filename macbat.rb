# macbat.rb

class Macbat < Formula
  desc "Утилита для мониторинга и управления зарядом батареи на ноутбуках с macOS"
  homepage "https://github.com/zeleza/macbat"
  url "https://github.com/zeleza/macbat/archive/refs/tags/v2.1.0.tar.gz"
  version "2.1.0"
  sha256 "76c31a3d2da8ebe4eafee5252b2db84afbe98b92265a3a6306edf2236fca544b"
  license "Apache-2.0"
  head "https://github.com/zeleza/macbat.git", branch: "main"

  # Зависимости для сборки
  depends_on "go" => :build
  depends_on xcode: :build  # Необходимо для CGO

  def install
    # Настройка переменных окружения для CGO
    ENV["CGO_ENABLED"] = "1"
    
    # Встраивание версии и даты сборки
    ldflags = %W[
      -s -w
      -X github.com/zeleza/macbat/internal/version.Version=#{version}
      -X github.com/zeleza/macbat/internal/version.BuildDate=#{time.iso8601}
    ]
    
    # Сборка из cmd/macbat
    system "go", "build", "-ldflags", ldflags.join(" "), 
           "-o", bin/"macbat", "./cmd/macbat"
  end

  def caveats
    <<~EOS
      Приложение macbat требует macOS и может потребовать дополнительных разрешений для мониторинга батареи.
      Вам может потребоваться предоставить доступ к приложению в System Preferences.
    EOS
  end

  test do
    # Базовый тест запуска
    assert_match version.to_s, shell_output("#{bin}/macbat --version 2>&1") || true
    
    # Проверка, что бинарник запускается
    system "#{bin}/macbat", "--help"
  end
end