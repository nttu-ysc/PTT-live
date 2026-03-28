package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/minio/selfupdate"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const AppVersion = "2.1.4"
const GitHubRepo = "nttu-ysc/PTT-live"

// App struct
type App struct {
	ctx           context.Context
	latestVersion string
	updateURL     string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// GetVersion returns the current app version string
func (a *App) GetVersion() string {
	return AppVersion
}

// OpenURL opens a URL in the system default browser
func (a *App) OpenURL(url string) {
	wailsruntime.BrowserOpenURL(a.ctx, url)
}

// CheckUpdate checks GitHub Releases API for a newer version.
// Returns a map with keys: hasUpdate (bool string), latestVersion, url
func (a *App) CheckUpdate() map[string]string {
	result := map[string]string{
		"hasUpdate":     "false",
		"latestVersion": AppVersion,
		"url":           fmt.Sprintf("https://github.com/%s/releases", GitHubRepo),
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", GitHubRepo)
	resp, err := http.Get(apiURL)
	if err != nil {
		return result
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return result
	}

	latestTag := strings.TrimPrefix(release.TagName, "v")
	currentTag := strings.TrimPrefix(AppVersion, "v")

	result["latestVersion"] = release.TagName
	result["url"] = release.HTMLURL

	if latestTag != currentTag {
		result["hasUpdate"] = "true"
		a.latestVersion = latestTag

		// Find suitable asset for auto-update
		for _, asset := range release.Assets {
			// For macOS (tar.gz)
			if runtime.GOOS == "darwin" && strings.HasSuffix(asset.Name, "mac-universal.tar.gz") {
				a.updateURL = asset.BrowserDownloadURL
				break
			}
			// For Windows (zip)
			if runtime.GOOS == "windows" && strings.HasSuffix(asset.Name, "windows-amd64.zip") {
				a.updateURL = asset.BrowserDownloadURL
				break
			}
		}
	}

	return result
}

// PerformUpdate downloads the update archive and applies it using selfupdate.
// It returns a success message or an error string.
func (a *App) PerformUpdate() string {
	if a.updateURL == "" {
		return "此平台尚無可用的自動更新檔案"
	}

	// 1. Download
	resp, err := http.Get(a.updateURL)
	if err != nil {
		return "下載更新失敗: " + err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("下載更新失敗 HTTP Status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "讀取更新檔失敗: " + err.Error()
	}

	// 2. Extract executable binary from archive
	var exeReader io.Reader

	if strings.HasSuffix(a.updateURL, ".tar.gz") {
		exeReader, err = extractTarGz(bodyBytes)
	} else if strings.HasSuffix(a.updateURL, ".zip") {
		exeReader, err = extractZip(bodyBytes)
	} else {
		return "未知的更新檔案格式"
	}

	if err != nil {
		return "解壓縮更新檔失敗: " + err.Error()
	}

	// 3. Apply selfupdate
	err = selfupdate.Apply(exeReader, selfupdate.Options{})
	if err != nil {
		return "套用更新檔失敗: " + err.Error()
	}

	// 4. macOS `.app` 簽名與版號修復
	// 當覆寫執行檔後，舊的 _CodeSignature 會與新的二進位不符，導致「檔案毀損」錯誤
	if runtime.GOOS == "darwin" {
		exe, err := os.Executable()
		if err == nil && strings.Contains(exe, ".app/Contents/MacOS/") {
			contentsDir := filepath.Dir(filepath.Dir(exe))

			// 刪除舊的 bundle signature
			sigDir := filepath.Join(contentsDir, "_CodeSignature")
			os.RemoveAll(sigDir)

			// 修改 Info.plist 中的版號為最新版
			plistPath := filepath.Join(contentsDir, "Info.plist")
			plistData, err := os.ReadFile(plistPath)
			if err == nil {
				reVersion := regexp.MustCompile(`(<key>CFBundleVersion</key>\s*<string>).*?(</string>)`)
				reShortVersion := regexp.MustCompile(`(<key>CFBundleShortVersionString</key>\s*<string>).*?(</string>)`)

				newData := reVersion.ReplaceAll(plistData, []byte("${1}"+a.latestVersion+"${2}"))
				newData = reShortVersion.ReplaceAll(newData, []byte("${1}"+a.latestVersion+"${2}"))

				os.WriteFile(plistPath, newData, 0644)
			}
		}
	}

	return "ok"
}

// Quit terminates the application so the user can restart it (for update completion)
func (a *App) Quit() {
	wailsruntime.Quit(a.ctx)
}

// Restart restarts the application by spawning a new instance of the current executable and quitting the current one.
func (a *App) Restart() {
	exe, err := os.Executable()
	if err == nil {
		if runtime.GOOS == "darwin" {
			if idx := strings.Index(exe, ".app/Contents/MacOS/"); idx != -1 {
				appPath := exe[:idx+4] // include ".app"
				cmd := exec.Command("open", "-n", appPath)
				cmd.Start()
			} else {
				cmd := exec.Command(exe)
				cmd.Start()
			}
		} else {
			cmd := exec.Command(exe)
			cmd.Start()
		}
	}
	a.Quit()
}

// extractTarGz extracts the PTT Live executable from the tar.gz payload
func extractTarGz(payload []byte) (io.Reader, error) {
	bytesReader := bytes.NewReader(payload)
	gzr, err := gzip.NewReader(bytesReader)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		// Expect the executable name to be exactly "PTT Live" (ignore macOS "._" hidden files)
		if header.Typeflag == tar.TypeReg && filepath.Base(header.Name) == "PTT Live" {
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, tr); err != nil {
				return nil, err
			}
			return &buf, nil
		}
	}
	return nil, errors.New("找不到執行檔於 tar.gz 中")
}

// extractZip extracts the PTT Live executable from the zip payload
func extractZip(payload []byte) (io.Reader, error) {
	bytesReader := bytes.NewReader(payload)
	zr, err := zip.NewReader(bytesReader, int64(len(payload)))
	if err != nil {
		return nil, err
	}

	for _, f := range zr.File {
		if !f.FileInfo().IsDir() && filepath.Base(f.Name) == "PTT Live.exe" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, rc); err != nil {
				return nil, err
			}
			return &buf, nil
		}
	}
	return nil, errors.New("找不到執行檔 \".exe\" 於 zip 中")
}
