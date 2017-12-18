package ltspice

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Raw reads an LTSpice raw (binary) file and returns a square matrix
// with the time in the first column, and data points in the rest.
// It also returns an array of column names
//
func Raw(file string) ([][]float64, []string, error) {

	var row, col int
	var fi *os.File
	var err error
	var vars []string
	var buf = []byte{0, 0}
	var lines []string
	var xvii = false

	// doubleVars is set if "Flags: double" is found (means that double's (64 bits)
	// are written to the raw file instead of floats (32 bits)
	doubleVars := false

	if len(file) == 0 {
		fi = os.Stdin
	} else {
		fi, err = os.Open(file)
		if err != nil {
			return nil, nil, err
		}
		defer fi.Close()
	}

	f := bufio.NewReader(fi)

	// First identify if the raw file comes from LTSpice IV or XVII
	//
	// LTSpice IV has an ASCII header, while LTSpice XVII uses UTF16LE

	i, err := f.Read(buf)

	if err != nil {
		return nil, nil, err
	}

	if i < 2 || buf[0] != 'T' {
		return nil, nil, errors.New("Not an LTSpice RAW file")
	}
	if buf[1] == 0 {
		xvii = true
	}

	// Read the text part

	if xvii {

		var uline []uint16

		for {
			n, _ := f.Read(buf)
			if n != 2 {
				return nil, nil, errors.New("premature end")
			}

			u := uint16(buf[0]) | uint16(buf[1]<<8)

			uline = append(uline, u)

			if buf[0] == 10 {

				line := string(utf16.Decode(uline))

				if line == "Binary:\n" {
					break
				}

				lines = append(lines, line)
				uline = nil

			}

		}

	} else {

		var b = []byte{0}
		var bb []byte

		for {
			n, _ := f.Read(b)
			if n != 1 {
				return nil, nil, errors.New("premature end")
			}

			bb = append(bb, b[0])

			if b[0] == 10 {

				line := string(bb)
				if line == "Binary:\n" {
					break
				}

				lines = append(lines, line)
				bb = nil
			}
		}

	}

	for i := 0; i < len(lines); i++ {

		s := lines[i]

		if strings.HasPrefix(s, "Flags:") {

			flags := strings.Split(s[6:], " ")

			for _, flag := range flags {
				if flag == "double" {
					doubleVars = true
					log.Println("double vars")
				} else if flag == "compressed" {
					return nil, nil, errors.New("compressed RAWs not supported")
				}
			}

		} else if strings.HasPrefix(s, "No. Variables:") {
			col, _ = strconv.Atoi(strings.TrimSpace(s[14:]))

		} else if strings.HasPrefix(s, "No. Points:") {
			row, _ = strconv.Atoi(strings.TrimSpace(s[11:]))

		} else if strings.HasPrefix(s, "Variables:") {

			for j := 0; j < col; j++ {
				i++
				s = lines[i]
				ss := strings.Split(s, "\t")
				vars = append(vars, ss[2])
			}
		}
	}

	// Read the binary part into an array

	Ma := make([]float64, row*col)

	// use slices to cover the array
	M := make([][]float64, col)
	for i := range M {
		M[i] = Ma[i*row : (i+1)*row]
	}

	t := make([]byte, 8)
	v4 := make([]byte, 4)
	v8 := make([]byte, 8)

	for i := 0; i < row; i++ {

		// Read a time stamp (8 bytes floating point)
		// [!] r.Read()  may not read len(buf) bytes even if there are left.
		_, err = io.ReadFull(f, t)

		if err != nil {
			log.Println("unexpected end")
			return nil, nil, err
		}
		M[0][i] = toFloat(t)

		for j := 1; j < col; j++ {

			if doubleVars {

				// Read a data point (8 bytes floating point)
				_, err = io.ReadFull(f, v8)

				if err != nil {
					log.Println("unexpected end")
					return nil, nil, err
				}

				M[j][i] = toFloat(v8)

			} else {

				// Read a data point (4 bytes floating point)
				_, err = io.ReadFull(f, v4)

				if err != nil {
					log.Println("unexpected end")
					return nil, nil, err
				}

				M[j][i] = toFloat32(v4)
			}
		}
	}

	return M, vars, nil
}

// toFloat converts an array of 8 bytes to a float64 value.
func toFloat(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)
	float := math.Float64frombits(bits)
	return float
}

// toFloat32 converts an array of 4 bytes to a float64 value.
func toFloat32(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint32(bytes)
	float := math.Float32frombits(bits)
	return float64(float)
}
