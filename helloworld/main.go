package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/tstpierre-tc/lcd1602"
	"periph.io/x/host/v3"
	//	"periph.io/x/conn/v3/i2c"
	//	"fmt"
	"periph.io/x/conn/v3/i2c/i2creg"
	"time"
)

var (
	busID             = "4"
	lcdAddress uint16 = 0x27
)

func main() {
	// Set up our peripherals
	if _, err := host.Init(); err != nil {
		log.Fatalf("Problem setting up periph host - %v", err)
	}
	log.Infof("Setting up i2c bus %s", busID)

	i2cbus, err := i2creg.Open(busID)
	if err != nil {
		log.Fatalf("Problem opening i2c bus %s - %v", busID, err)
	}
	defer i2cbus.Close()

	lcd, lcdErr := lcd1602.NewI2C(i2cbus, &lcd1602.Opts{
		I2CAddr:   lcdAddress,
		Lines:     2,
		Cols:      16,
		CharDelay: 100 * time.Millisecond,
	})
	if lcdErr != nil {
		log.Fatalf("Problem opening display - %v", err)
	}
	defer lcd.Halt()
	log.Info("Turning on the backlight")
	lcd.SetBacklight(true)

	//	lcd.SetPosition(1, 6)
	check := []byte{
		0b00000,
		0b00000,
		0b00001,
		0b00010,
		0b10100,
		0b01000,
		0b00000,
		0b00000,
	}
	sig1 := []byte{
		0b00000,
		0b00000,
		0b00000,
		0b00000,
		0b00000,
		0b00000,
		0b10000,
		0b10000,
	}

	sig2 := []byte{
		0b00000,
		0b00000,
		0b00000,
		0b00000,
		0b00100,
		0b00100,
		0b10100,
		0b10100,
	}

	sig3 := []byte{
		0b00000,
		0b00001,
		0b00001,
		0b00001,
		0b00101,
		0b00101,
		0b10101,
		0b10101,
	}
	bell := []byte{
		0b00000,
		0b00100,
		0b01110,
		0b01110,
		0b01110,
		0b11111,
		0b00100,
		0b00000,
	}

	err = lcd.SetCustomChar("checkmark", check)
	if err != nil {
		log.Error(err)
	}
	err = lcd.SetCustomChar("signal1", sig1)
	if err != nil {
		log.Error(err)
	}
	err = lcd.SetCustomChar("signal2", sig2)
	if err != nil {
		log.Error(err)
	}
	err = lcd.SetCustomChar("signal3", sig3)
	if err != nil {
		log.Error(err)
	}
	err = lcd.SetCustomChar("bell", bell)
	if err != nil {
		log.Error(err)
	}

	lcd.Clear()
	lcd.Home()
	lcd.Write([]byte("Test"))
	lcd.CustomChar("checkmark")
	lcd.CustomChar("signal1")
	lcd.CustomChar("signal2")
	lcd.CustomChar("signal3")
	lcd.CustomChar("bell")
	if err != nil {
		log.Error(err)
	}

	/*
		lcd.Write([]byte("Test 1234"))
		time.Sleep(time.Second)
		lcd.SetPosition(2, 6)
		lcd.Write([]byte("Test 1234"))
		time.Sleep(time.Second)
		lcd.SetDisplayShift(true)
		lcd.Write([]byte("Test 5678"))
	*/
	/*

		if err := lcd.SetPosition(1, lcd.Right()); err != nil {
			log.Error(err)
		}

		lcd.SetDisplayShift(true)
	*/
	/*
		for i := 0; i < 127; i++ {
			lcd.Write([]byte(fmt.Sprintf(" %d", i)))
			time.Sleep(200 * time.Millisecond)
		}
	*/
	//	lcd.SetIncrement(true)

	/*
		lcd.Clear()
		lcd.SetPosition(1, 7)
		lcd.Write([]byte("Test 1234 1234 1234 1234"))
		time.Sleep(5 * time.Second)
		for i := 0; i < 10; i++ {
			lcd.SetDisplayShift(true)
			time.Sleep(100 * time.Millisecond)
		}
		time.Sleep(5 * time.Second)
		lcd.SetDisplayShift(true)
		time.Sleep(5 * time.Second)
		lcd.SetShiftRight(true)
		time.Sleep(5 * time.Second)

		lcd.Write([]byte("Test abc"))
		time.Sleep(5 * time.Second)
	*/
	/*
		lcd.WriteCell('a')
		lcd.WriteCell('b')
		lcd.WriteCell('c')
		lcd.WriteCell('d')
	*/
	time.Sleep(5 * time.Second)
	log.Info("Turning off the backlight")
	lcd.SetBacklight(false)
}
