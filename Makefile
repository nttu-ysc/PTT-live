# ─────────────────────────────────────────────────────────────────────────────
# PTT Live – Build Makefile
#
# 需求工具:
#   - wails       (https://wails.io)
#   - mingw-w64   (build windows): brew install mingw-w64
#   - hdiutil     已內建於 macOS，不需額外安裝
#
# 使用方式:
#   make dmg        # Universal DMG (arm64 + amd64 合一，推薦)
#   make mac-arm64  # 只 build Apple Silicon .app
#   make mac-amd64  # 只 build Intel .app
#   make windows    # Windows .exe (需要 mingw-w64)
#   make all        # 所有平台
#   make clean      # 清除 build 輸出
# ─────────────────────────────────────────────────────────────────────────────

APP      := PTT Live
BIN_DIR  := build/bin
DMG_DIR  := build/dmg

# 暫存路徑（避免 -clean 把另一個 arch 的輸出刪掉）
STAGE_ARM64 := /tmp/ptt-live-arm64.app
STAGE_AMD64 := /tmp/ptt-live-amd64.app

# 最終輸出
APP_UNIV := $(BIN_DIR)/$(APP).app
DMG_OUT  := $(DMG_DIR)/$(APP).dmg

.PHONY: all mac-arm64 mac-amd64 universal windows dmg clean help

# ── Default ──────────────────────────────────────────────────────────────────
all: universal windows

# ── macOS Apple Silicon ───────────────────────────────────────────────────────
mac-arm64:
	@echo "▶ Building macOS Apple Silicon (darwin/arm64)…"
	wails build -platform darwin/arm64 -clean
	@rm -rf "$(STAGE_ARM64)"
	@cp -R "$(BIN_DIR)/$(APP).app" "$(STAGE_ARM64)"
	@echo "✅ arm64 暫存於 $(STAGE_ARM64)"

# ── macOS Intel ───────────────────────────────────────────────────────────────
mac-amd64:
	@echo "▶ Building macOS Intel (darwin/amd64)…"
	wails build -platform darwin/amd64 -clean
	@rm -rf "$(STAGE_AMD64)"
	@cp -R "$(BIN_DIR)/$(APP).app" "$(STAGE_AMD64)"
	@echo "✅ amd64 暫存於 $(STAGE_AMD64)"

# ── Universal Binary (arm64 + amd64) ─────────────────────────────────────────
universal: mac-arm64 mac-amd64
	@echo "▶ Creating Universal Binary with lipo…"
	@mkdir -p "$(BIN_DIR)"
	@rm -rf "$(APP_UNIV)"
	@cp -R "$(STAGE_ARM64)" "$(APP_UNIV)"
	lipo -create \
		"$(STAGE_ARM64)/Contents/MacOS/$(APP)" \
		"$(STAGE_AMD64)/Contents/MacOS/$(APP)" \
		-output "$(APP_UNIV)/Contents/MacOS/$(APP)"
	@echo "▶ Stripping old code signatures (prevent 'unsealed contents' error)…"
	@# Remove _CodeSignature dirs (stores the actual signature data)
	@find "$(APP_UNIV)" -name "_CodeSignature" -type d -exec rm -rf {} + 2>/dev/null || true
	@# Remove code-signing extended attributes
	@xattr -cr "$(APP_UNIV)" 2>/dev/null || true
	@echo "▶ Ad-hoc signing (繞過「無法確認開發者」提示)…"
	codesign --deep --force --options runtime --sign - "$(APP_UNIV)"
	@echo "✅ Universal: $(APP_UNIV)"

# ── Windows ───────────────────────────────────────────────────────────────────
# 需要: brew install mingw-w64
windows:
	@echo "▶ Building Windows (windows/amd64)…"
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
		CC=x86_64-w64-mingw32-gcc \
		wails build -platform windows/amd64 -clean
	@echo "✅ $(BIN_DIR)/$(APP).exe"

# ── DMG (Universal) ───────────────────────────────────────────────────────────
# 使用內建 hdiutil 打包，不需安裝額外工具
# Universal Binary 同時包含 arm64 + amd64，一個 DMG 即可
dmg: universal
	@echo "▶ Packaging Universal DMG…"
	@mkdir -p "$(DMG_DIR)"
	@rm -f "$(DMG_OUT)"
	# 建立暫存目錄放 .app 和 Applications 捷徑
	@rm -rf /tmp/ptt-dmg-stage
	@mkdir /tmp/ptt-dmg-stage
	@cp -R "$(APP_UNIV)" /tmp/ptt-dmg-stage/
	@ln -s /Applications "/tmp/ptt-dmg-stage/Applications"
	# 打包成壓縮 DMG
	hdiutil create \
		-volname "$(APP)" \
		-srcfolder /tmp/ptt-dmg-stage \
		-ov -format UDZO \
		"$(DMG_OUT)"
	@rm -rf /tmp/ptt-dmg-stage
	@echo "✅ DMG: $(DMG_OUT)"
	@echo ""
	@echo "  ⚠️  首次開啟若出現「無法確認開發者」，右鍵 → 開啟；或執行:"
	@echo "       xattr -rd com.apple.quarantine /Applications/PTT\\ Live.app"

# ── Clean ─────────────────────────────────────────────────────────────────────
clean:
	rm -rf build/bin build/dmg
	@echo "✅ Cleaned"

# ── Help ──────────────────────────────────────────────────────────────────────
help:
	@echo ""
	@echo "  make dmg        – Universal DMG (arm64 + amd64, 推薦)"
	@echo "  make universal  – Universal .app (arm64 + amd64 合一)"
	@echo "  make mac-arm64  – Apple Silicon .app"
	@echo "  make mac-amd64  – Intel Mac .app"
	@echo "  make windows    – Windows .exe    (需要 brew install mingw-w64)"
	@echo "  make all        – Universal + Windows"
	@echo "  make clean      – 清除 build/"
	@echo ""
