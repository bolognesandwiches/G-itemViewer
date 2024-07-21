package main

import (
	"archive/zip"
	"context"
	"embed"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

const (
	HWND_TOP       uintptr = 0
	SWP_SHOWWINDOW         = 0x0040
	GWL_STYLE              = -16
	GWL_EXSTYLE            = -20
	WS_CAPTION             = 0x00C00000
	WS_THICKFRAME          = 0x00040000
	WS_SYSMENU             = 0x00080000
)

//go:embed all:frontend/dist
var assets embed.FS

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procSetParent                = user32.NewProc("SetParent")
	procSetWindowLong            = user32.NewProc("SetWindowLongW")
	procGetWindowLong            = user32.NewProc("GetWindowLongW")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procGetClientRect            = user32.NewProc("GetClientRect")
)

type App struct {
	ctx       context.Context
	habboHWND syscall.Handle
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Quit() {
	runtime.Quit(a.ctx)
}

func intptr(n int) uintptr {
	return uintptr(n)
}

func (a *App) DownloadAndExtractZip(url string) (string, error) {
	runtime.LogInfo(a.ctx, "Starting download from: "+url)
	resp, err := http.Get(url)
	if err != nil {
		runtime.LogError(a.ctx, "Download failed: "+err.Error())
		return "", err
	}
	defer resp.Body.Close()

	runtime.LogInfo(a.ctx, "Creating temporary file")
	tmpfile, err := os.CreateTemp("", "download-*.zip")
	if err != nil {
		runtime.LogError(a.ctx, "Failed to create temp file: "+err.Error())
		return "", err
	}
	defer os.Remove(tmpfile.Name())

	runtime.LogInfo(a.ctx, "Copying download to temp file")
	_, err = io.Copy(tmpfile, resp.Body)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to copy download: "+err.Error())
		return "", err
	}

	runtime.LogInfo(a.ctx, "Opening zip file")
	r, err := zip.OpenReader(tmpfile.Name())
	if err != nil {
		runtime.LogError(a.ctx, "Failed to open zip: "+err.Error())
		return "", err
	}
	defer r.Close()

	tempDir, err := os.MkdirTemp("", "habbo-extract-")
	if err != nil {
		runtime.LogError(a.ctx, "Failed to create temp directory: "+err.Error())
		return "", err
	}

	runtime.LogInfo(a.ctx, "Extracting contents to: "+tempDir)
	for _, f := range r.File {
		runtime.LogInfo(a.ctx, "Extracting file: "+f.Name)

		if !strings.HasPrefix(f.Name, "26 - Copy/") {
			continue
		}

		extractPath := filepath.Join(tempDir, strings.TrimPrefix(f.Name, "26 - Copy/"))

		if f.FileInfo().IsDir() {
			os.MkdirAll(extractPath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(extractPath), os.ModePerm); err != nil {
			runtime.LogError(a.ctx, "Failed to create directory: "+err.Error())
			return "", err
		}

		outFile, err := os.OpenFile(extractPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			runtime.LogError(a.ctx, "Failed to create file: "+err.Error())
			return "", err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			runtime.LogError(a.ctx, "Failed to open file in zip: "+err.Error())
			return "", err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			runtime.LogError(a.ctx, "Failed to write file: "+err.Error())
			return "", err
		}
	}

	runtime.LogInfo(a.ctx, "Extraction completed successfully")
	return tempDir, nil
}

func (a *App) RunExecutable(path string) error {
	runtime.LogInfo(a.ctx, "Running executable: "+path)
	cmd := exec.Command(path)
	cmd.Dir = filepath.Dir(path)
	return cmd.Start()
}

func (a *App) LaunchHabbo() error {
	url := "https://github.com/bolognesandwiches/G-Inventory-Viewer/raw/master/assets/26.zip"

	runtime.LogInfo(a.ctx, "Starting Habbo launch process")
	extractPath, err := a.DownloadAndExtractZip(url)
	if err != nil {
		return err
	}

	execPath := filepath.Join(extractPath, "Habbo.exe")
	runtime.LogInfo(a.ctx, "Attempting to run: "+execPath)
	return a.RunExecutable(execPath)
}

func (a *App) EmbedHabboWindow(habboHWND syscall.Handle) error {
	// Find the Wails window by its title
	a.habboHWND = habboHWND
	wailsHWND, err := FindWailsWindow("Habbo Embed App") // Make sure this matches your window title
	if err != nil {
		return err
	}

	// Remove border and make non-moveable
	currentStyle, _, _ := procGetWindowLong.Call(uintptr(habboHWND), intptr(GWL_STYLE))
	procSetWindowLong.Call(
		uintptr(habboHWND),
		intptr(GWL_STYLE),
		currentStyle & ^uintptr(WS_CAPTION) & ^uintptr(WS_THICKFRAME) & ^uintptr(WS_SYSMENU),
	)

	// Set the Habbo window as a child of the Wails window
	procSetParent.Call(uintptr(habboHWND), uintptr(wailsHWND))

	// Get the size of the Wails window
	var wailsRect windows.Rect
	procGetClientRect.Call(uintptr(wailsHWND), uintptr(unsafe.Pointer(&wailsRect)))

	// Define the size of the Habbo window
	habboWidth := int32(720)
	habboHeight := int32(540)

	// Calculate the position to center the Habbo window
	x := (wailsRect.Right - wailsRect.Left - habboWidth) / 2
	y := (wailsRect.Bottom - wailsRect.Top - habboHeight) / 2

	// Resize and position the Habbo window
	procSetWindowPos.Call(
		uintptr(habboHWND),
		uintptr(HWND_TOP),
		uintptr(x),
		uintptr(y),
		uintptr(habboWidth),
		uintptr(habboHeight),
		uintptr(SWP_SHOWWINDOW),
	)

	runtime.LogInfo(a.ctx, fmt.Sprintf("Habbo window centered at (%d, %d) with size %dx%d", x, y, habboWidth, habboHeight))

	return nil
}

func (a *App) OnResize(width int, height int) {
	if a.habboHWND != 0 {
		err := a.UpdateHabboWindowPosition(a.habboHWND)
		if err != nil {
			runtime.LogError(a.ctx, fmt.Sprintf("Failed to update Habbo window position: %v", err))
		}
	}
}

func (a *App) WaitForHabboWindow() {
	for i := 0; i < 30; i++ { // Try for 30 seconds
		var habboHandle syscall.Handle
		cb := windows.NewCallback(func(h syscall.Handle, p uintptr) uintptr {
			var title [200]uint16
			_, _, _ = procGetWindowTextW.Call(uintptr(h), uintptr(unsafe.Pointer(&title[0])), 200)
			if windows.UTF16ToString(title[:]) == "Habbo Hotel: Origins" {
				habboHandle = h
				return 0
			}
			return 1
		})
		procEnumWindows.Call(cb, 0)

		if habboHandle != 0 {
			runtime.LogInfo(a.ctx, fmt.Sprintf("Found Habbo window after %d seconds", i))
			err := a.EmbedHabboWindow(habboHandle) // Pass habboHandle as an argument
			if err != nil {
				runtime.LogError(a.ctx, fmt.Sprintf("Failed to embed Habbo window: %v", err))
			}
			return
		}

		time.Sleep(1 * time.Second)
	}
	runtime.LogError(a.ctx, "Failed to find Habbo window after 30 seconds")
}

func (a *App) LaunchAndEmbedHabbo() string {
	runtime.LogInfo(a.ctx, "Starting LaunchAndEmbedHabbo process")

	err := a.DownloadAndLaunchHabbo() // Make sure this line is present
	if err != nil {
		errMsg := "Failed to launch and embed Habbo: " + err.Error()
		runtime.LogError(a.ctx, errMsg)
		return errMsg
	}

	return "Habbo launched and embedded successfully. Check the application window."
}

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "Habbo Embed App",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 0},
		Frameless:        true,
		OnStartup:        app.startup,

		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

func (a *App) HandleResize() {
	if a.habboHWND != 0 {
		err := a.UpdateHabboWindowPosition(a.habboHWND)
		if err != nil {
			runtime.LogError(a.ctx, fmt.Sprintf("Failed to update Habbo window position: %v", err))
		}
	}
}

func (a *App) DownloadAndLaunchHabbo() error {
	// Download the zip file
	zipPath := filepath.Join(os.TempDir(), "habbo.zip")
	err := downloadFile("https://github.com/bolognesandwiches/G-Inventory-Viewer/raw/master/assets/26.zip", zipPath)
	if err != nil {
		return fmt.Errorf("Failed to download Habbo: %v", err)
	}

	// Unpack the zip file
	tempDir, err := ioutil.TempDir("", "habbo")
	if err != nil {
		return fmt.Errorf("Failed to create temp directory: %v", err)
	}
	err = unzip(zipPath, tempDir)
	if err != nil {
		return fmt.Errorf("Failed to unpack Habbo: %v", err)
	}

	// Find the Habbo.exe file
	var habboPath string
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "Habbo.exe" {
			habboPath = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Error searching for Habbo.exe: %v", err)
	}
	if habboPath == "" {
		return fmt.Errorf("Habbo.exe not found in the extracted files")
	}

	// Launch Habbo
	cmd := exec.Command(habboPath)
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("Failed to launch Habbo: %v", err)
	}

	// Wait a bit for Habbo to launch
	time.Sleep(5 * time.Second)

	// Find the Habbo window
	habboHWND, err := FindWindowByProcess(cmd.Process.Pid)
	if err != nil {
		return fmt.Errorf("Failed to find Habbo window: %v", err)
	}

	// Embed the Habbo window
	err = a.EmbedHabboWindow(habboHWND)
	if err != nil {
		return fmt.Errorf("Failed to embed Habbo window: %v", err)
	}

	return nil
}

func FindWindowByProcess(pid int) (syscall.Handle, error) {
	var hwnd syscall.Handle
	cb := syscall.NewCallback(func(h syscall.Handle, param uintptr) uintptr {
		var processID uint32
		procGetWindowThreadProcessId.Call(uintptr(h), uintptr(unsafe.Pointer(&processID)))
		if int(processID) == pid {
			hwnd = h
			return 0 // stop enumeration
		}
		return 1 // continue enumeration
	})
	procEnumWindows.Call(cb, 0)
	if hwnd == 0 {
		return 0, fmt.Errorf("window not found for process ID %d", pid)
	}
	return hwnd, nil
}
func FindWailsWindow(title string) (syscall.Handle, error) {
	var hwnd syscall.Handle
	cb := syscall.NewCallback(func(h syscall.Handle, param uintptr) uintptr {
		var buf [256]uint16
		procGetWindowTextW.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), 256)
		if syscall.UTF16ToString(buf[:]) == title {
			hwnd = h
			return 0 // stop enumeration
		}
		return 1 // continue enumeration
	})
	procEnumWindows.Call(cb, 0)
	if hwnd == 0 {
		return 0, fmt.Errorf("Wails window with title '%s' not found", title)
	}
	return hwnd, nil
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *App) UpdateHabboWindowPosition(habboHWND syscall.Handle) error {
	wailsHWND, err := FindWailsWindow("Habbo Embed App") // Make sure this matches your window title
	if err != nil {
		return err
	}

	var wailsRect windows.Rect
	procGetClientRect.Call(uintptr(wailsHWND), uintptr(unsafe.Pointer(&wailsRect)))

	habboWidth := int32(720)
	habboHeight := int32(540)

	x := (wailsRect.Right - wailsRect.Left - habboWidth) / 2
	y := (wailsRect.Bottom - wailsRect.Top - habboHeight) / 2

	procSetWindowPos.Call(
		uintptr(habboHWND),
		uintptr(HWND_TOP),
		uintptr(x),
		uintptr(y),
		uintptr(habboWidth),
		uintptr(habboHeight),
		uintptr(SWP_SHOWWINDOW),
	)

	runtime.LogInfo(a.ctx, fmt.Sprintf("Habbo window repositioned to (%d, %d)", x, y))

	return nil
}
