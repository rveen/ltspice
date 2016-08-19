package ltspice

import (
	"bufio"
	"encoding/binary"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

// Raw reads an LTSpice raw (binary) file and returns a square matrix
// with the time in the first column, and data points in the rest.
// It also returns an array of column names
func Raw(file string, header bool) ([][]float64, []string, error) {

	var row, col int
	var f *os.File
	var err error
	var vars []string

	if len(file) == 0 {
		f = os.Stdin
	} else {
		f, err = os.Open(file)
		if err != nil {
			return nil, nil, err
		}
		defer f.Close()
	}

	r := bufio.NewReader(f)

	// Read text part
	for {

		s, err := r.ReadString('\n')
		if err != nil || len(s) == 0 {
			break
		}

		if strings.HasPrefix(s, "No. Variables:") {
			col, _ = strconv.Atoi(strings.TrimSpace(s[14:]))

		} else if strings.HasPrefix(s, "No. Points:") {
			row, _ = strconv.Atoi(strings.TrimSpace(s[11:]))

		} else if strings.HasPrefix(s, "Variables:") {

			for j := 0; j < col; j++ {
				s, err = r.ReadString('\n')
				if err != nil || len(s) == 0 {
					break
				}
				ss := strings.Split(s, "\t")
				vars = append(vars, ss[2])
			}
		} else if strings.HasPrefix(s, "Binary:") {
			break
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
	v := make([]byte, 4)

	for i := 0; i < row; i++ {

		// Read a time stamp (8 bytes floating point)
		// [!] r.Read()  may not read len(buf) bytes even if there are left.
		_, err = io.ReadFull(r, t)

		if err != nil {
			log.Println("unexpected end")
			return nil, nil, err
		}
		M[0][i] = toFloat(t)

		for j := 1; j < col; j++ {

			// Read a data point (4 bytes floating point)
			_, err = io.ReadFull(r, v)

			if err != nil {
				log.Println("unexpected end")
				return nil, nil, err
			}
			M[j][i] = toFloat32(v)
		}
	}

	return M, vars, nil
}

// toFloat converts an array of 8 bytes to a
// float64 value.
func toFloat(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)
	float := math.Float64frombits(bits)
	return float
}

// toFloat32 converts an array of 4 bytes to a
// float64 value.
func toFloat32(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint32(bytes)
	float := math.Float32frombits(bits)
	return float64(float)
}
