package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/GaryBoone/GoStats/stats"
	"github.com/rveen/ltspice"
)

// lta - LT data analyzer

func main() {

	var summary bool
	var duty int

	flag.IntVar(&duty, "d", 0, "Calculate duty cycle of the specified column")
	flag.BoolVar(&summary, "s", false, "Print summary")
	flag.Parse()

	file := ""
	if len(flag.Args()) > 0 {
		file = flag.Args()[0]
	}

	m, vars, err := ltspice.Raw(file, false)

	if err != nil {
		log.Println(err)
	}

	if m == nil {
		log.Println("null matrix")
	}

	cols := len(m)
	rows := len(m[0])

	if duty == 0 {
		for j := 1; j < cols; j++ {

			mean := stats.StatsMean(m[j])
			sdev := stats.StatsSampleStandardDeviation(m[j])

			fmt.Printf("%-20s %30g %30g %30g\n", "'"+vars[j]+"'", mean, sdev, sdev/mean)
		}
	} else {

		var dcs []float64

		//fmt.Printf("duty of %s\n", vars[duty])
		min := stats.StatsMin(m[duty])
		max := stats.StatsMax(m[duty])

		mid := (max-min)/2 + min
		//fmt.Printf("min %f max %f, rows %d, thres %f\n", min, max, rows, mid)

		ni := 0
		nf := 0

		for i := 0; i < rows; i++ {

			// detect LT runs (time = 0)
			if i > 0 && m[0][i] == 0 {
				nf = i - 1
				i++

				// Calculate DC
				m, _ := Dutycycle(m[duty][ni:nf], m[0][ni:nf], mid)
				dcs = append(dcs, m)

				ni = i
			}
		}

		ni = nf + 2
		nf = rows - 1

		// Calculate DC
		m, _ := Dutycycle(m[duty][ni:nf], m[0][ni:nf], mid)
		dcs = append(dcs, m)

		if summary {
			mean := stats.StatsMean(dcs)
			max = stats.StatsMax(dcs)
			min = stats.StatsMin(dcs)
			fmt.Printf("mean %f, min %f, max %f\n", mean, min, max)
		}

	}
}

func Dutycycle(a []float64, t []float64, mid float64) (float64, float64) {
	// fmt.Printf("--------------------------\ndc %d len mid %f\n", len(a), mid)

	// Detect transitions

	// initial state = low
	state := false

	if a[0] > mid {
		state = true
	}

	var transitions []int

	for i := 1; i < len(a); i++ {
		switch state {
		case false:
			// detect transition to high
			if a[i] > mid {
				//fmt.Println("low to high @", t[i])
				state = true
				transitions = append(transitions, i)
			}

		case true:
			// detect transition to low
			if a[i] < mid {
				//fmt.Println("high to low @", t[i])
				state = false
				transitions = append(transitions, -i)
			}

		}
	}

	ti := 0.0
	tf := 0.0

	tdiff0 := 0.0

	var dcs []float64
	var dc float64

	for i, tr := range transitions {

		n := tr
		if n < 0 {
			n = -n
		}

		if i == 0 {
			ti = t[n]
			continue
		}

		tf = t[n]

		/*
			if tr < 0 {
				fmt.Printf("HL %d t %f\n", tr, tf-ti)
			} else {
				fmt.Printf("LH %d t %f\n", tr, tf-ti)
			}*/

		if i == 1 {
			continue
		}

		tdiff := tf - ti
		if i&1 == 1 {

			if tr < 0 {
				dc = tdiff / (tdiff + tdiff0)
			} else {
				dc = 1.0 - tdiff/(tdiff+tdiff0)
			}

			dcs = append(dcs, dc)

		}

		tdiff0 = tf - ti
		ti = tf
	}

	mean := stats.StatsMean(dcs)
	std := stats.StatsSampleStandardDeviation(dcs)

	//fmt.Printf("mean %f, std %f\n", mean, std)

	// Clean dcs
	var dcc []float64
	for _, dc = range dcs {
		if dc > mean+std || dc < mean-std {
			continue
		}
		dcc = append(dcc, dc)
	}

	mean = stats.StatsMean(dcc)
	std = stats.StatsSampleStandardDeviation(dcc)

	fmt.Printf("%f, %f\n", mean, std)

	return mean, std
}
