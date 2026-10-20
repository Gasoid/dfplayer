// Package dfplayer provides a Go driver for the DFPlayer Mini MP3 module
// over UART (serial). It supports sending commands, receiving responses,
// and handling common playback operations.
package dfplayer

import (
	"io"
	"time"
)

const (
	bufSize    uint8  = 10
	packetSize int    = 10
	noParam    uint16 = 0x00

	startByte    uint8 = 0x7E
	readTimeout        = 500 * time.Millisecond
	pollInterval       = time.Millisecond
)

// DPCode represents a DFPlayer command or response code.
type DPCode uint8

type errorParam uint8

// Response codes sent by the DFPlayer.
const (
	TrackFinishedCode DPCode = 0x3D + iota
	_
	ReadyCode
	ErrorCode
)

// Command codes for controlling the DFPlayer.
const (
	_ DPCode = iota
	NextTrackCode
	PreviousTrackCode
	PlayRootFolderCode
	IncreaseVolumeCode
	DecreaseVolumeCode
	SetVolumeCode
	SetEQCode
	SetRepeatModeCode
	SelectSourceCode
	SleepCode
	WakeUpCode
	ResetCode
	ResumeCode
	PauseCode
	PlayFolderCode
	GainCode
	LoopCode
	PlayMp3FolderCode
	PlayAdvertiseCode
	PlayLargeFolderCode
	StopAdvertiseCode
	StopCode
	RepeatFolderCode
	RandomCode
	RepeatCurrentCode
	DACCode
)

const (
	StateCode DPCode = 0x42 + iota
	VolumeCode
	EqCode
	PlaybackModeCode
	SoftwareVersionCode
	TotalFilesUDiskCode
	TotalFilesSDCode
	TotalFilesFlashCode
	// 0x4A is "Keep on" — undocumented, no known function.
)

const (
	CurrentTrackUDiskCode DPCode = 0x4B + iota
	CurrentTrackSDCode
	CurrentTrackFlashCode
	FileCountsInFolderCode
	TotalFoldersCode
)

const (
	_ errorParam = iota
	busy
	sleeping
	serialWrongStack
	checkSumNotMatch
	fileIndexOut
	fileMismatch
	advertise
)

type Message struct {
	Cmd    DPCode
	ParamH uint8
	ParamL uint8
}

// Param returns the full 16-bit parameter value (ParamH<<8 | ParamL).
func (m *Message) Param() uint16 {
	return uint16(m.ParamH)<<8 | uint16(m.ParamL)
}

// readUart reads a 10-byte packet from the UART reader with a 500ms deadline.
// It synchronizes on the start byte (0x7E) and returns the full packet.
// When the reader implements bufferedReader (e.g. machine.UART), reads are
// non-blocking and the deadline is strictly honored. Otherwise it falls back
// to blocking reads and assumes the reader manages its own timeout.
func readUart(r io.Reader) ([]byte, error) {
	deadline := time.Now().Add(readTimeout)
	got := make([]byte, 0, packetSize)
	var buf [bufSize]byte

	for len(got) < packetSize {
		if time.Now().After(deadline) {
			return nil, &ErrUartRead{text: "read timeout"}
		}

		n, err := readAvailable(r, buf[:])
		if err != nil {
			return nil, &ErrUartRead{text: "Couldn't read from UART", err: err}
		}

		if n == 0 {
			time.Sleep(pollInterval)
			continue
		}

		if len(got) > 0 {
			got = append(got, buf[:n]...)
		} else {
			for i := range buf[:n] {
				if buf[i] == startByte {
					got = append(got, buf[i:n]...)
					break
				}
			}
		}
	}

	return got, nil
}

// bufferedReader is implemented by readers that can report how many bytes are
// available to read without blocking. TinyGo's machine.UART and bufio.Reader
// both implement this.
type bufferedReader interface {
	Buffered() int
}

// readAvailable reads up to len(buf) bytes without blocking longer than
// necessary. If r implements bufferedReader, it reads only the bytes known
// to be available and returns (0, nil) when the buffer is empty.
// Otherwise it performs a plain blocking read.
func readAvailable(r io.Reader, buf []byte) (int, error) {
	br, ok := r.(bufferedReader)
	if !ok {
		return r.Read(buf)
	}

	avail := br.Buffered()
	if avail == 0 {
		return 0, nil
	}

	if avail > len(buf) {
		avail = len(buf)
	}
	return r.Read(buf[:avail])
}

// Dfplayer represents a connected DFPlayer Mini module.
type Dfplayer struct {
	uart io.ReadWriter
}

// New creates a new Dfplayer instance using the provided UART ReadWriter.
func New(uart io.ReadWriter) (*Dfplayer, error) {
	return &Dfplayer{uart}, nil
}

// ReadMessage reads and returns the next command/response code from the DFPlayer.
func (d *Dfplayer) ReadMessage() (*Message, error) {
	data, err := readUart(d.uart)
	if err != nil {
		return nil, err
	}

	if !isChecksumValid(data) {
		return nil, &ErrDfplayer{"checksum is invalid"}
	}

	if ErrorCode == DPCode(data[3]) {
		switch errorParam(data[6]) {
		case busy:
			return nil, ErrBusy
		case sleeping:
			return nil, ErrSleeping
		case serialWrongStack:
			return nil, ErrSerialWrongStack
		case checkSumNotMatch:
			return nil, ErrCheckSumNotMatch
		case fileIndexOut:
			return nil, ErrFileIndexOut
		case fileMismatch:
			return nil, ErrFileMismatch
		case advertise:
			return nil, ErrAdvertise
		default:
			return nil, &ErrDfplayer{"dfplayer returns error, but error is unknown"}
		}
	}

	msg := Message{
		Cmd:    DPCode(data[3]),
		ParamH: data[5],
		ParamL: data[6],
	}

	return &msg, nil
}

// IsTrackFinished checks whether the currently playing track has finished.
func (d *Dfplayer) IsTrackFinished() (bool, error) {
	msg, err := d.ReadMessage()
	if err != nil {
		return false, err
	}

	return TrackFinishedCode == msg.Cmd, nil
}

// WaitTrack blocks until a track finishes or the timeout expires.
// Typical usage after play:
//
//	if err := player.WaitTrack(5 * time.Second); err != nil {
//	    println("dfplayer error:", err.Error())
//	}
func (d *Dfplayer) WaitTrack(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		finished, _ := d.IsTrackFinished()
		if finished {
			return nil
		}

		if time.Now().After(deadline) {
			return ErrTimeout
		}
	}
}

// IsReady checks whether the DFPlayer is ready to accept commands.
// Call this after power-on before sending playback commands.
func (d *Dfplayer) IsReady() (bool, error) {
	msg, err := d.ReadMessage()
	if err != nil {
		return false, err
	}

	return ReadyCode == msg.Cmd, nil
}

// WaitReady blocks until the DFPlayer is ready or the timeout expires.
// Typical usage after power-on:
//
//	if err := player.WaitReady(5 * time.Second); err != nil {
//	    log.Fatal("DFPlayer not ready:", err)
//	}
func (d *Dfplayer) WaitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		ready, _ := d.IsReady()
		if ready {
			return nil
		}

		if time.Now().After(deadline) {
			return ErrTimeout
		}
	}
}

// SendCmd sends a command with a 16-bit parameter to the DFPlayer.
func (d *Dfplayer) SendCmd(cmd DPCode, param uint16) error {
	return sendMessage(d.uart, cmd, param)
}

// sendMessage writes a properly formatted DFPlayer packet to the writer.
// Packet layout:
//
//	0: Start(0x7E) | 1: Version(0xFF) | 2: Length(0x06) | 3: Command
//	4: Feedback(0x00) | 5: Param_H | 6: Param_L | 7: Checksum_H | 8: Checksum_L | 9: End(0xEF)
func sendMessage(w io.Writer, cmd DPCode, param uint16) error {
	packet := []byte{
		0x7E, 0xFF, 0x06, byte(cmd), 0x00,
		byte(param >> 8),
		byte(param & 0xFF),
		0x00, 0x00,
		0xEF,
	}

	checksum := calcChecksum(packet)

	packet[7] = byte(checksum >> 8)
	packet[8] = byte(checksum & 0xFF)

	if _, err := w.Write(packet); err != nil {
		return &ErrUartWrite{text: "Couldn't write to UART", err: err}
	}

	return nil
}

// calcChecksum computes the DFPlayer checksum over bytes 1–6.
// Checksum = -(sum of bytes 1..6) as a 16-bit two's complement value.
func calcChecksum(packet []byte) uint16 {
	var sum uint16
	for i := 1; i <= 6; i++ {
		sum += uint16(packet[i])
	}

	return uint16(0 - sum)
}

// isChecksumValid verifies the checksum in a received DFPlayer packet.
func isChecksumValid(packet []byte) bool {
	if len(packet) < 10 {
		return false
	}

	checksum := calcChecksum(packet)

	h := byte(checksum >> 8)
	l := byte(checksum & 0xFF)

	return packet[7] == h && packet[8] == l
}
