package dfplayer

const (
	maxVolume   uint16 = 30
	enableParam uint16 = iota
	disableParam
)

// PlayMp3Track plays a track from the "mp3" folder on the SD card.
// Track must be in range 1–2999.
func (d *Dfplayer) PlayMp3Track(track uint16) error {
	if track == 0 || track > 2999 {
		return &ErrDfplayer{"wrong track number, track is 0 or greater than 2999"}
	}

	return d.SendCmd(PlayMp3FolderCode, track)
}

// PlayRootTrack plays a track from the root folder of the storage device.
// Track must be in range 1–2999.
func (d *Dfplayer) PlayRootTrack(track uint16) error {
	if track == 0 || track > 2999 {
		return &ErrDfplayer{"wrong track number, track is 0 or greater than 2999"}
	}
	return d.SendCmd(PlayRootFolderCode, track)
}

// NextTrack skips to the next track.
func (d *Dfplayer) NextTrack() error {
	return d.SendCmd(NextTrackCode, noParam)
}

// PreviousTrack goes back to the previous track.
func (d *Dfplayer) PreviousTrack() error {
	return d.SendCmd(PreviousTrackCode, noParam)
}

// VolumeUp increases the volume by 1. Maximum volume is 30.
func (d *Dfplayer) VolumeUp() error {
	return d.SendCmd(IncreaseVolumeCode, noParam)
}

// VolumeDown decreases the volume by 1.
func (d *Dfplayer) VolumeDown() error {
	return d.SendCmd(DecreaseVolumeCode, noParam)
}

// SetVolume sets the volume to a specific level. Maximum volume is 30.
func (d *Dfplayer) SetVolume(vol uint16) error {
	if vol > maxVolume {
		return &ErrDfplayer{"vol can't be greater than 30"}
	}

	return d.SendCmd(SetVolumeCode, vol)
}

// EqMode represents an equalizer preset.
type EqMode uint16

// Available equalizer presets.
const (
	EqNormal  EqMode = iota // Normal
	EqPop                   // Pop
	EqRock                  // Rock
	EqJazz                  // Jazz
	EqClassic               // Classic
	EqBass                  // Bass
)

// SetEQ applies an equalizer preset.
func (d *Dfplayer) SetEQ(eq EqMode) error {
	return d.SendCmd(SetEQCode, uint16(eq))
}

// Sleep puts the DFPlayer into low-power sleep mode.
func (d *Dfplayer) Sleep() error {
	return d.SendCmd(SleepCode, noParam)
}

// WakeUp wakes the DFPlayer from sleep mode.
func (d *Dfplayer) WakeUp() error {
	return d.SendCmd(WakeUpCode, noParam)
}

// Reset performs a software reset of the DFPlayer.
func (d *Dfplayer) Reset() error {
	return d.SendCmd(ResetCode, noParam)
}

// Resume resumes playback after a pause.
func (d *Dfplayer) Resume() error {
	return d.SendCmd(ResumeCode, noParam)
}

// Pause pauses the current playback.
func (d *Dfplayer) Pause() error {
	return d.SendCmd(PauseCode, noParam)
}

// Stop stops playback.
func (d *Dfplayer) Stop() error {
	return d.SendCmd(StopCode, noParam)
}

// Advertise plays an advertisement track from the "advert" folder.
// Track must be in range 1–2999.
func (d *Dfplayer) Advertise(track uint16) error {
	if track == 0 || track > 2999 {
		return &ErrDfplayer{"wrong track number, track is 0 or greater than 2999"}
	}

	return d.SendCmd(PlayAdvertiseCode, track)
}

// StopAdvertise stops the currently playing advertisement.
func (d *Dfplayer) StopAdvertise() error {
	return d.SendCmd(StopAdvertiseCode, noParam)
}

// PlayLargeFolder plays a track from a large folder (supports up to 3000 tracks).
// Folder must be in range 1–15, track must be in range 1–3000.
func (d *Dfplayer) PlayLargeFolder(folder, track uint16) error {
	if folder == 0 || folder > 15 {
		return &ErrDfplayer{"wrong folder, folder is 0 or greater than 15"}
	}

	if track == 0 || track > 3000 {
		return &ErrDfplayer{"wrong track number, track is 0 or greater than 3000"}
	}

	param := (folder << 12) | track
	return d.SendCmd(PlayLargeFolderCode, param)
}

// EnableDAC enables the DAC output (turns audio output on).
func (d *Dfplayer) EnableDAC() error {
	return d.SendCmd(DACCode, enableParam)
}

// DisableDAC disables the DAC output (turns audio output off).
func (d *Dfplayer) DisableDAC() error {
	return d.SendCmd(DACCode, disableParam)
}

// EnableLoopAll enables looping of all tracks.
func (d *Dfplayer) EnableLoopAll() error {
	return d.SendCmd(LoopCode, 0x01)
}

// DisableLoopAll disables looping of all tracks.
func (d *Dfplayer) DisableLoopAll() error {
	return d.SendCmd(LoopCode, 0x00)
}

// PlayFolder plays a track from a specific folder.
// Folder must be in range 1–99, track must be in range 1–255.
func (d *Dfplayer) PlayFolder(folder, track uint16) error {
	if folder == 0 || folder > 99 {
		return &ErrDfplayer{"wrong folder, folder is 0 or greater than 99"}
	}

	if track == 0 || track > 255 {
		return &ErrDfplayer{"wrong track number, track is 0 or greater than 255"}
	}

	param := (folder << 8) | track
	return d.SendCmd(PlayFolderCode, param)
}

// RepeatMode represents the playback repeat mode.
type RepeatMode uint16

// Available repeat modes.
const (
	RepeatModeAll    RepeatMode = iota // Repeat all tracks continuously
	RepeatModeFolder                   // Repeat all tracks in the current folder
	RepeatModeOne                      // Repeat the current track
	RepeatModeRandom                   // Play tracks in random order
)

// SetRepeatMode sets the playback repeat mode.
// Note: support for this command depends on the DFPlayer firmware variant.
func (d *Dfplayer) SetRepeatMode(mode RepeatMode) error {
	return d.SendCmd(SetRepeatModeCode, uint16(mode))
}

// Source represents the playback source (storage device).
type Source uint16

// Available playback sources.
const (
	_           Source = iota
	SourceUSB          // USB flash drive
	SourceSD           // microSD card
	SourceAUX          // AUX/line-in input
	SourceSleep        // Sleep mode
	SourceFlash        // On-chip flash memory
)

// SelectSource switches the active playback source.
func (d *Dfplayer) SelectSource(source Source) error {
	return d.SendCmd(SelectSourceCode, uint16(source))
}

// SetOutput configures the DAC output and gain level.
// enable turns the output on (true) or off (false).
// gain sets the gain level in range 0–31.
func (d *Dfplayer) SetOutput(enable bool, gain uint16) error {
	if gain > 31 {
		return &ErrDfplayer{"gain can't be greater than 31"}
	}
	var param uint16
	if enable {
		param = 0x0100
	}
	param |= gain
	return d.SendCmd(GainCode, param)
}

// RepeatFolder loops all tracks in the specified folder.
// Folder must be in range 1–99.
func (d *Dfplayer) RepeatFolder(folder uint16) error {
	if folder == 0 || folder > 99 {
		return &ErrDfplayer{"wrong folder, folder is 0 or greater than 99"}
	}

	return d.SendCmd(RepeatFolderCode, folder)
}

// RandomAll plays all tracks across all folders in random (shuffle) order.
func (d *Dfplayer) RandomAll() error {
	return d.SendCmd(RandomCode, noParam)
}

// EnableRepeatCurrent enables single-track repeat (loops the current track).
func (d *Dfplayer) EnableRepeatCurrent() error {
	return d.SendCmd(RepeatCurrentCode, enableParam)
}

// DisableRepeatCurrent disables single-track repeat.
func (d *Dfplayer) DisableRepeatCurrent() error {
	return d.SendCmd(RepeatCurrentCode, disableParam)
}

// Drain discards any pending bytes from the UART receive buffer without
// blocking. It is useful to drop stale events (for example, right after
// WaitReady, or before reading a known event such as IsTrackFinished).
//
// Drain is a no-op if the underlying reader does not implement the
// bufferedReader interface (Buffered() int), since there is no safe,
// non-blocking way to drain such a reader.
func (d *Dfplayer) Drain() error {
	br, ok := d.uart.(bufferedReader)
	if !ok {
		return nil
	}

	var buf [bufSize]byte
	for br.Buffered() > 0 {
		n := min(br.Buffered(), len(buf))
		read, err := d.uart.Read(buf[:n])
		if err != nil {
			return &ErrUartRead{text: "Couldn't read from UART", err: err}
		}

		if read == 0 {
			break
		}
	}

	return nil
}

// State represents the playback status reported by the DFPlayer.
// Note: the DFPlayer hardware only reports two states — playing or not playing.
// There is no separate "paused" state; after pausing, the module reports stopped.
type State uint8

const (
	StateStopped State = iota
	StatePlaying
)

// GetState queries the current playback status.
// Returns StatePlaying (1) when a track is active, StateStopped (0) otherwise.
func (d *Dfplayer) GetState() (State, error) {
	if err := d.SendCmd(StateCode, noParam); err != nil {
		return 0, err
	}

	msg, err := d.ReadMessage()
	if err != nil {
		return 0, err
	}

	if msg.Cmd == StateCode {
		return State(msg.ParamL), nil
	}

	return 0, ErrUnexpectedCode
}

// GetVolume queries the current volume.
func (d *Dfplayer) GetVolume() (uint16, error) {
	if err := d.SendCmd(VolumeCode, noParam); err != nil {
		return 0, err
	}

	msg, err := d.ReadMessage()
	if err != nil {
		return 0, err
	}

	if msg.Cmd == VolumeCode {
		return uint16(msg.ParamL), nil
	}

	return 0, ErrUnexpectedCode
}

// GetEq queries the current equalizer mode.
func (d *Dfplayer) GetEq() (EqMode, error) {
	if err := d.SendCmd(EqCode, noParam); err != nil {
		return 0, err
	}

	msg, err := d.ReadMessage()
	if err != nil {
		return 0, err
	}

	if msg.Cmd == EqCode {
		return EqMode(msg.ParamL), nil
	}

	return 0, ErrUnexpectedCode
}

// GetPlaybackMode queries the current playback mode.
// Note: the meaning of the returned value is firmware-dependent and of limited use.
func (d *Dfplayer) GetPlaybackMode() (uint16, error) {
	if err := d.SendCmd(PlaybackModeCode, noParam); err != nil {
		return 0, err
	}

	msg, err := d.ReadMessage()
	if err != nil {
		return 0, err
	}

	if msg.Cmd == PlaybackModeCode {
		return msg.Param(), nil
	}

	return 0, ErrUnexpectedCode
}

// GetSoftwareVersion queries the DFPlayer firmware version number.
func (d *Dfplayer) GetSoftwareVersion() (uint16, error) {
	if err := d.SendCmd(SoftwareVersionCode, noParam); err != nil {
		return 0, err
	}

	msg, err := d.ReadMessage()
	if err != nil {
		return 0, err
	}

	if msg.Cmd == SoftwareVersionCode {
		return uint16(msg.ParamL), nil
	}

	return 0, ErrUnexpectedCode
}

// GetTotalFiles returns the total number of files on the specified storage source.
// Supported sources: SourceUSB, SourceSD, SourceFlash.
func (d *Dfplayer) GetTotalFiles(source Source) (uint16, error) {
	code, err := totalFilesCode(source)
	if err != nil {
		return 0, err
	}

	if err := d.SendCmd(code, noParam); err != nil {
		return 0, err
	}

	msg, err := d.ReadMessage()
	if err != nil {
		return 0, err
	}

	if msg.Cmd == code {
		return msg.Param(), nil
	}

	return 0, ErrUnexpectedCode
}

// GetCurrentTrack returns the number of the currently playing track on the specified source.
// Supported sources: SourceUSB, SourceSD, SourceFlash.
func (d *Dfplayer) GetCurrentTrack(source Source) (uint16, error) {
	code, err := currentTrackCode(source)
	if err != nil {
		return 0, err
	}

	if err := d.SendCmd(code, noParam); err != nil {
		return 0, err
	}

	msg, err := d.ReadMessage()
	if err != nil {
		return 0, err
	}

	if msg.Cmd == code {
		return msg.Param(), nil
	}

	return 0, ErrUnexpectedCode
}

// GetFileCountsInFolder returns the number of files in the specified folder (1–99).
func (d *Dfplayer) GetFileCountsInFolder(folder uint16) (uint16, error) {
	if folder == 0 || folder > 99 {
		return 0, &ErrDfplayer{"wrong folder, folder is 0 or greater than 99"}
	}

	if err := d.SendCmd(FileCountsInFolderCode, folder); err != nil {
		return 0, err
	}

	msg, err := d.ReadMessage()
	if err != nil {
		return 0, err
	}

	if msg.Cmd == FileCountsInFolderCode {
		return uint16(msg.ParamL), nil
	}

	return 0, ErrUnexpectedCode
}

// GetTotalFolders returns the total number of folders on the current storage device,
// including the root directory.
func (d *Dfplayer) GetTotalFolders() (uint16, error) {
	if err := d.SendCmd(TotalFoldersCode, noParam); err != nil {
		return 0, err
	}

	msg, err := d.ReadMessage()
	if err != nil {
		return 0, err
	}

	if msg.Cmd == TotalFoldersCode {
		return uint16(msg.ParamL), nil
	}

	return 0, ErrUnexpectedCode
}

// totalFilesCode maps a storage source to the corresponding query command.
func totalFilesCode(source Source) (DPCode, error) {
	switch source {
	case SourceUSB:
		return TotalFilesUDiskCode, nil
	case SourceSD:
		return TotalFilesSDCode, nil
	case SourceFlash:
		return TotalFilesFlashCode, nil
	default:
		return 0, &ErrDfplayer{"unsupported source for GetTotalFiles: use SourceUSB, SourceSD, or SourceFlash"}
	}
}

// currentTrackCode maps a storage source to the corresponding query command.
func currentTrackCode(source Source) (DPCode, error) {
	switch source {
	case SourceUSB:
		return CurrentTrackUDiskCode, nil
	case SourceSD:
		return CurrentTrackSDCode, nil
	case SourceFlash:
		return CurrentTrackFlashCode, nil
	default:
		return 0, &ErrDfplayer{"unsupported source for GetCurrentTrack: use SourceUSB, SourceSD, or SourceFlash"}
	}
}
