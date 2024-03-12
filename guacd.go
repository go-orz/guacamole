package guacamole

import (
	"fmt"
)

const (
	Username = "username"
	Password = "password"

	EnableRecording     = "enable-recording"
	RecordingPath       = "recording-path"
	CreateRecordingPath = "create-recording-path"

	FontName     = "font-name"
	FontSize     = "font-size"
	ColorScheme  = "color-scheme"
	Backspace    = "backspace"
	TerminalType = "terminal-type"

	PreConnectionId   = "preconnection-id"
	PreConnectionBlob = "preconnection-blob"

	DisableAudio     = "disable-audio"
	EnableAudioInput = "enable-audio-input"

	EnableDrive     = "enable-drive"
	DriveName       = "drive-name"
	DrivePath       = "drive-path"
	CreateDrivePath = "create-drive-path"

	Security     = "security"
	IgnoreCert   = "ignore-cert"
	ResizeMethod = "resize-method"

	EnablePrinting = "enable-printing"
	PrinterName    = "printer-name"
	PrinterDriver  = "printer-driver"

	EnableWallpaper          = "enable-wallpaper"
	EnableTheming            = "enable-theming"
	EnableFontSmoothing      = "enable-font-smoothing"
	EnableFullWindowDrag     = "enable-full-window-drag"
	EnableDesktopComposition = "enable-desktop-composition"
	EnableMenuAnimations     = "enable-menu-animations"
	DisableBitmapCaching     = "disable-bitmap-caching"
	DisableOffscreenCaching  = "disable-offscreen-caching"
	// DisableGlyphCaching Deprecated
	DisableGlyphCaching = "disable-glyph-caching"
	ForceLossless       = "force-lossless"

	Domain        = "domain"
	RemoteApp     = "remote-app"
	RemoteAppDir  = "remote-app-dir"
	RemoteAppArgs = "remote-app-args"

	ColorDepth  = "color-depth"
	Cursor      = "cursor"
	SwapRedBlue = "swap-red-blue"
	DestHost    = "dest-host"
	DestPort    = "dest-port"
	ReadOnly    = "read-only"

	UsernameRegex     = "username-regex"
	PasswordRegex     = "password-regex"
	LoginSuccessRegex = "login-success-regex"
	LoginFailureRegex = "login-failure-regex"

	Namespace  = "namespace"
	Pod        = "pod"
	Container  = "container"
	UesSSL     = "use-ssl"
	ClientCert = "client-cert"
	ClientKey  = "client-key"
	CaCert     = "ca-cert"
)

var recordingInst = []string{
	"cfill",
	"size",
	"move",
	"blob",
	"end",
	"cursor",
	"sync",
	"copy",
	"rect",
	"img",
	"dispose",
	"audio",
	"mouse",
}

type Configuration struct {
	ConnectionID string
	Protocol     string
	Parameters   map[string]string
}

func NewConfiguration() (config *Configuration) {
	config = &Configuration{}
	config.Parameters = make(map[string]string)
	return config
}

func (opt *Configuration) SetReadOnlyMode() {
	opt.Parameters[ReadOnly] = "true"
}

func (opt *Configuration) SetParameter(name, value string) {
	opt.Parameters[name] = value
}

func (opt *Configuration) UnSetParameter(name string) {
	delete(opt.Parameters, name)
}

func (opt *Configuration) GetParameter(name string) string {
	return opt.Parameters[name]
}

type Instruction struct {
	Opcode       string
	Args         []string
	ProtocolForm string
}

func NewInstruction(opcode string, args ...string) *Instruction {
	instruction := Instruction{
		Opcode:       opcode,
		Args:         args,
		ProtocolForm: "",
	}
	return &instruction
}

func (opt *Instruction) String() string {
	if len(opt.ProtocolForm) > 0 {
		return opt.ProtocolForm
	}

	opt.ProtocolForm = fmt.Sprintf("%d.%s", len(opt.Opcode), opt.Opcode)
	for _, value := range opt.Args {
		opt.ProtocolForm += fmt.Sprintf(",%d.%s", len(value), value)
	}
	opt.ProtocolForm += string(Delimiter)
	return opt.ProtocolForm
}
