KOTLINC="${KOTLINC:-kotlinc}"
if ! command -v "$KOTLINC" > /dev/null 2>&1; then
  echo "kotlinc not found. Set KOTLINC=/path/to/kotlinc"
  exit 1
fi
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
mkdir -p "$SCRIPT_DIR/build"
"$KOTLINC" "$SCRIPT_DIR/tep/src/main/kotlin" "$SCRIPT_DIR/tep/src/test/kotlin" \
  -include-runtime -d "$SCRIPT_DIR/build/tep-tests.jar" > /dev/null 2>&1
if [ $? -ne 0 ]; then
  echo "build failed"
  exit 1
fi
echo "built $SCRIPT_DIR/build/tep-tests.jar"

if [ "$1" = "run" ]; then
  java -cp "$SCRIPT_DIR/build/tep-tests.jar" uz.asad.tep.TepTestKt
fi