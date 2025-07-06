#!/usr/bin/env bash
# update_build_number.sh
# Скрипт авто-инкрементирует номер сборки для текущей версии.
# Использование: ./scripts/update_build_number.sh <version>

set -euo pipefail

VERSION=${1:-dev}
COUNTER_FILE=".build_number"

# Если версия изменилась, сбрасываем счётчик
if [[ -f "$COUNTER_FILE" ]]; then
  read -r CURRENT_VERSION CURRENT_BUILD < "$COUNTER_FILE"
else
  CURRENT_VERSION=""
  CURRENT_BUILD="0"
fi

if [[ "$VERSION" != "$CURRENT_VERSION" ]]; then
  BUILD_NUMBER=1
else
  BUILD_NUMBER=$((CURRENT_BUILD + 1))
fi

echo "$VERSION $BUILD_NUMBER" > "$COUNTER_FILE"

echo "$BUILD_NUMBER"
