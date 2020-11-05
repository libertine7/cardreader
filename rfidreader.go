package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/jacobsa/go-serial/serial"
)

var currentCard string
var lock sync.RWMutex

func createRequest(command []byte, param []byte) []byte {
	data := make([]byte, 0, 100)
	data = append(data, []byte{0x00, 0x00}...) // device id
	data = append(data, command...)            // command
	if len(param) > 0 {
		data = append(data, param...)
	}
	newData := make([]byte, 0, 100)

	for _, one := range data {
		if one == 0xAA {
			newData = append(newData, []byte{0xAA, 0x00}...)

		} else {
			newData = append(newData, one)
		}
	}

	result := make([]byte, 0, 100)
	result = append(result, []byte{0xAA, 0xBB}...)             // head
	result = append(result, []byte{byte(len(data) + 1), 0}...) // length
	result = append(result, newData...)                        // data
	result = append(result, verifyData(newData))               // verify

	return result
}

func verifyData(data []byte) byte {
	var verificationXor byte
	verificationXor = 0x00
	for _, one := range data {
		verificationXor = verificationXor ^ one
	}
	return verificationXor
}

func sendCommand(ser io.ReadWriteCloser, command []byte) (byte, []byte, error) {
	n, err := ser.Write(command)
	if err != nil {
		return 0, nil, err
	}
	buf := make([]byte, 128)
	n, err = ser.Read(buf)
	if err != nil {
		return 0, nil, err
	}

	buf = buf[:n]

	if !bytes.HasPrefix(buf, []byte{0xAA, 0xBB}) {
		return 0, nil, errors.New("wrong prefix")
	}

	dataLen := buf[2]

	// fmt.Printf("*buf  [% x]\n", buf[:len(buf)])
	buf = bytes.ReplaceAll(buf, []byte{0xAA, 0x00}, []byte{0xAA})

	if buf[len(buf)-1] != verifyData(buf[4:dataLen+3]) {
		return 0, nil, errors.New("wrong crc")
	}

	return buf[8], buf[9 : dataLen+3], nil // state, data, err
}

func readCard(ser io.ReadWriteCloser) (string, error) {
	cardRaw := []byte{}

	_, _, err := sendCommand(ser, createRequest([]byte{0x01, 0x01}, []byte{0x03})) // init
	if err != nil {
		return "", err
	}

	_, _, err = sendCommand(ser, createRequest([]byte{0x07, 0x01}, []byte{0x00})) // light off
	if err != nil {
		return "", err
	}

	state, data, err := sendCommand(ser, createRequest([]byte{0x01, 0x02}, []byte{0x52})) // request card type
	if err == nil && state == 0 {
		if data[0] == 2 || data[0] == 4 {
			_, cardRaw, err = sendCommand(ser, createRequest([]byte{0x02, 0x02}, []byte{})) // Anticollision
		} else {
			_, cardRaw, err = sendCommand(ser, createRequest([]byte{0x12, 0x02}, []byte{})) // Anticollision
		}
	}
	if err != nil {
		return "", err
	}

	_, _, err = sendCommand(ser, createRequest([]byte{0x07, 0x01}, []byte{0x02})) // light green

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%X", cardRaw), nil
}

func checkPort(port string) error {
	ser, err := serial.Open(serial.OpenOptions{PortName: port, BaudRate: 19200, DataBits: 8, StopBits: 1, MinimumReadSize: 1})
	if err != nil {
		return err
	}
	defer ser.Close()

	_, _, err = sendCommand(ser, createRequest([]byte{0x01, 0x01}, []byte{0x03})) // init
	if err != nil {
		return err
	}

	return nil
}

func findPort() string {
	ports := []string{"/dev/ttyUSB0", "/dev/ttyUSB1", "/dev/ttyUSB2", "/dev/ttyUSB3", "/dev/ttyUSB4", "/dev/tty.SLAB_USBtoUART"}
	for _, port := range ports {
		err := checkPort(port)
		if err == nil {
			return port
		}
	}
	return ""
}

func CurrentCard() string {
	lock.RLock()
	card := currentCard
	lock.RUnlock()
	return card
}

func setCard(card string) {
	lock.Lock()
	currentCard = card
	lock.Unlock()
}

func ReadCards(broadcast chan string) {
	for {
		port := findPort()
		if port != "" {
			s, err := serial.Open(serial.OpenOptions{PortName: port, BaudRate: 19200, DataBits: 8, StopBits: 1, MinimumReadSize: 1})
			if err != nil {
				break
			}

			for {
				card, err := readCard(s)

				if err != nil {
					break
				}
				if card != CurrentCard() {
					setCard(card)
					broadcast <- card
				}

				time.Sleep(100 * time.Millisecond)
			}
			s.Close()
		}
		setCard("")
		time.Sleep(5 * time.Second)
	}
}
