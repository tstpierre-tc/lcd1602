/*
Copyright 2024 Tim St. Pierre
Controls a 1602 character LCD display using I2C backpack
Thanks to Dave Cheney for figuring out the registers!
*/
package lcd1602

import (
	"encoding/binary"

	"fmt"

	log "github.com/sirupsen/logrus"
	"periph.io/x/conn/v3"
	"time"
	//	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/mmr"
)

const (
	// Commands
	CMD_Clear_Display        = 0x01
	CMD_Return_Home          = 0x02
	CMD_Entry_Mode           = 0x04
	CMD_Display_Control      = 0x08
	CMD_Cursor_Display_Shift = 0x10
	CMD_Function_Set         = 0x20
	CMD_CGRAM_Set            = 0x40
	CMD_DDRAM_Set            = 0x80

	// Options
	OPT_Increment      = 0x02 // CMD_Entry_Mode
	OPT_Cursor_Shift   = 0x01 // CMD_Entry_Mode
	OPT_Enable_Display = 0x04 // CMD_Display_Control
	OPT_Enable_Cursor  = 0x02 // CMD_Display_Control
	OPT_Enable_Blink   = 0x01 // CMD_Display_Control
	OPT_Display_Shift  = 0x08 // CMD_Cursor_Display_Shift
	OPT_Shift_Right    = 0x04 // CMD_Cursor_Display_Shift 0 = Left
	OPT_2_Lines        = 0x08 // CMD_Function_Set 0 = 1 line
	OPT_5x10_Dots      = 0x04 // CMD_Function_Set 0 = 5x7 dots

	// Pins
	EN        = 2
	WR        = 1
	RS        = 0
	D4        = 4
	D5        = 5
	D6        = 6
	D7        = 7
	BACKLIGHT = 3
)

type Dev struct {
	isSPI           bool
	displayEnable   bool
	backlight_state bool
	cursor          bool
	blink           bool
	displayShift    bool
	shiftRight      bool
	c               mmr.Dev8
	Opts            Opts
	nextCustomIndex uint8
	CharMap         map[string]uint8
}

func (d *Dev) String() string {
	return fmt.Sprintf("lcd1602{%s}", d.c.Conn)
}

// NewI2C returns a new device that communicates over I²C
//
// Use default options if nil is used.
func NewI2C(b i2c.Bus, opts *Opts) (*Dev, error) {
	if opts == nil {
		opts = &DefaultOpts
	}
	addr, err := opts.i2cAddr()
	if err != nil {
		return nil, fmt.Errorf("lcd1602 %x: %v", addr, err)
	}
	d, err := makeDev(&i2c.Dev{Bus: b, Addr: addr}, false, opts)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// Halt is a noop for the cap1xxx.
func (d *Dev) Halt() error {
	d.Clear()
	d.SetBacklight(false)
	return nil
}

func (d *Dev) SetBacklight(on bool) {
	d.c.WriteUint8(0, pinInterpret(BACKLIGHT, 0x00, on))
	d.backlight_state = on
}

func (d *Dev) Clear() {
	d.command(CMD_Clear_Display)
}

func (d *Dev) Home() {
	d.command(CMD_Return_Home)
}

func (d *Dev) SetPosition(line, pos byte) error {
	if line > d.Opts.Lines {
		return fmt.Errorf("lcd1602 %x: device does not support %d lines", line, d.Opts.I2CAddr)
	}
	if pos > d.Opts.Cols {
		return fmt.Errorf("lcd1602 %x: device does not support %d cols", line, d.Opts.I2CAddr)
	}
	var address byte
	switch line {
	case 1:
		address = pos
	case 2:
		address = 0x40 + pos
	case 3:
		address = 0x10 + pos
	case 4:
		address = 0x50 + pos
	}
	d.command(CMD_DDRAM_Set + address)
	return nil
}

func (d *Dev) Write(buf []byte) (int, error) {
	for _, c := range buf {
		d.write(c, false)
		time.Sleep(d.Opts.CharDelay)
	}
	log.Printf("%s %d", buf, buf)
	return len(buf), nil
}

func (d *Dev) WriteChar(char byte) error {
	d.write(char, false)
	d.CursorShift(false)
	return nil
}

func (d *Dev) Right() byte {
	return d.Opts.Cols
}

func (d *Dev) SetCustomChar(name string, data []byte) error {
	if d.nextCustomIndex >= 7 {
		return fmt.Errorf("lcd1602 %x: No more custome character space! ", d.Opts.I2CAddr)
	}
	if len(data) != 8 {
		return fmt.Errorf("lcd1602 %x: Custom character must be 8 bytes! ", d.Opts.I2CAddr)
	}
	address := CMD_CGRAM_Set | (d.nextCustomIndex * 8)
	log.Infof("Writing to address %b - %x", address, address)
	d.write(address, true)
	for _, v := range data {
		d.write(v, false)
	}
	d.CharMap[name] = d.nextCustomIndex
	d.nextCustomIndex++
	return nil
}

func (d *Dev) CustomChar(name string) error {
	addr, hasChar := d.CharMap[name]
	if !hasChar {
		return fmt.Errorf("lcd1602 %x: Custom character %s not set! ", d.Opts.I2CAddr, name)
	}
	d.write(addr, false)
	return nil
}

func makeDev(c conn.Conn, isSPI bool, opts *Opts) (*Dev, error) {
	d := &Dev{
		displayEnable: true,
		cursor:        true,
		blink:         true,
		displayShift:  false,
		shiftRight:    false,
		Opts:          *opts,
		isSPI:         isSPI,
		c:             mmr.Dev8{Conn: c, Order: binary.LittleEndian},
		CharMap:       make(map[string]uint8),
	}

	// Activate LCD
	var data byte
	data = pinInterpret(D4, data, true)
	data = pinInterpret(D5, data, true)
	d.enable(data)
	time.Sleep(200 * time.Millisecond)
	d.enable(data)
	time.Sleep(100 * time.Millisecond)
	d.enable(data)
	time.Sleep(100 * time.Millisecond)

	// Initialize 4-bit mode
	data = pinInterpret(D4, data, false)
	d.enable(data)
	time.Sleep(10 * time.Millisecond)

	d.command(CMD_Function_Set | OPT_2_Lines)
	// d.command(CMD_Display_Control | OPT_Enable_Display)
	d.writeDisplaySwitch()
	d.writeEntryMode()
	d.command(CMD_Clear_Display)
	return d, nil
}

func (d *Dev) SetDisplayShift(value bool) {
	d.displayShift = value
	d.writeEntryMode()
}

func (d *Dev) SetShiftRight(value bool) {
	d.shiftRight = value
	d.writeEntryMode()
}

func (d *Dev) writeDisplaySwitch() {
	option := byte(CMD_Display_Control)
	if d.displayEnable {
		option = option | OPT_Enable_Display
	}
	if d.cursor {
		option = option | OPT_Enable_Cursor
	}
	if d.blink {
		option = option | OPT_Enable_Blink
	}
	log.Debug("Writing display switch")
	d.command(option)
}

func (d *Dev) DisplayShift(right bool) {
	option := byte(CMD_Cursor_Display_Shift | OPT_Display_Shift)
	if right {
		option = option | OPT_Shift_Right
	}
	d.command(option)
}

func (d *Dev) CursorShift(right bool) {
	log.Debug("Writing cursor shift")
	option := byte(CMD_Cursor_Display_Shift)
	if right {
		option = option | OPT_Shift_Right
	}
	d.command(option)
}

func (d *Dev) CursorOn(on bool) {
	d.cursor = on
	d.writeDisplaySwitch()
}

func (d *Dev) SetBlink(on bool) {
	d.blink = on
	d.writeDisplaySwitch()
}

func (d *Dev) writeEntryMode() {
	option := byte(CMD_Entry_Mode)
	if !d.shiftRight {
		option = option | OPT_Increment
	}
	if d.displayShift {
		option = option | OPT_Cursor_Shift
	}
	d.command(option)
}

func (d *Dev) command(data byte) {
	d.write(data, true)
	time.Sleep(100 * time.Microsecond)
}

func (d *Dev) WriteCell(char byte) {
	d.write(0x40|char, false)
}
func (d *Dev) write(data byte, command bool) {
	var i2c_data byte
	log.Debugf("Writing %b %x", data, data)
	// Add data for high nibble
	hi_nibble := data >> 4
	i2c_data = pinInterpret(D4, i2c_data, (hi_nibble&0x01 == 0x01))
	i2c_data = pinInterpret(D5, i2c_data, ((hi_nibble>>1)&0x01 == 0x01))
	i2c_data = pinInterpret(D6, i2c_data, ((hi_nibble>>2)&0x01 == 0x01))
	i2c_data = pinInterpret(D7, i2c_data, ((hi_nibble>>3)&0x01 == 0x01))

	// # Set the register selector to 1 if this is data
	if !command {
		i2c_data = pinInterpret(RS, i2c_data, true)
	}

	//  Toggle Enable
	d.enable(i2c_data)

	i2c_data = 0x00

	// Add data for high nibble
	low_nibble := data & 0x0F
	i2c_data = pinInterpret(D4, i2c_data, (low_nibble&0x01 == 0x01))
	i2c_data = pinInterpret(D5, i2c_data, ((low_nibble>>1)&0x01 == 0x01))
	i2c_data = pinInterpret(D6, i2c_data, ((low_nibble>>2)&0x01 == 0x01))
	i2c_data = pinInterpret(D7, i2c_data, ((low_nibble>>3)&0x01 == 0x01))

	// Set the register selector to 1 if this is data
	if !command {
		i2c_data = pinInterpret(RS, i2c_data, true)
	}

	d.enable(i2c_data)
}

func (d *Dev) enable(data byte) {
	// Determine if black light is on and insure it does not turn off or on
	if d.backlight_state {
		data = pinInterpret(BACKLIGHT, data, true)
	} else {
		data = pinInterpret(BACKLIGHT, data, false)
	}
	d.c.WriteUint8(0, data)
	time.Sleep(40 * time.Microsecond)
	d.c.WriteUint8(0, pinInterpret(EN, data, true))
	time.Sleep(40 * time.Microsecond)
	d.c.WriteUint8(0, data)
}

// Still don't completely understand this - hope to soon
func pinInterpret(pin, data byte, value bool) byte {
	if value {
		// Construct mask using pin
		var mask byte = 0x01 << (pin)
		data = data | mask
	} else {
		// Construct mask using pin
		var mask byte = 0x01<<(pin) ^ 0xFF
		data = data & mask
	}
	return data
}
