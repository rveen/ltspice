package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/GaryBoone/GoStats/stats"
	"github.com/atgjack/prob"
	"github.com/rveen/ltspice"
)

// lta - LT data analyzer
//
// TODO support options measdgt, numdgt
// TODO support LTSpice XVII format

type Parameter struct {
	Name     string
	Max      float64
	Min      float64
	Mean     float64
	StdDev   float64
	Cpk      float64
	Ppm      float64
	Good     float64
	MaxHere  float64
	MinHere  float64
	MaxCount int
	MinCount int
}

var Parameters []Parameter

var upperLimit = 0.0
var lowerLimit = 0.0

func main() {

	var summary, header, data, csv bool
	var duty, hist, rms int

	flag.IntVar(&duty, "d", 0, "Calculate duty cycle of the specified column")
	flag.IntVar(&rms, "rms", 0, "Calculate the RMS value of the specified column")
	flag.IntVar(&hist, "hist", 0, "Generate histogram (hist.svg)")
	flag.BoolVar(&summary, "s", false, "Print summary")
	flag.BoolVar(&data, "data", false, "Print raw data (use with -rms)")
	flag.BoolVar(&header, "v", false, "Print header")
	flag.BoolVar(&csv, "csv", false, "Print CSV")
	flag.Float64Var(&upperLimit, "max", 0.0, "Establish the upper limit of the parameter under study")
	flag.Float64Var(&lowerLimit, "min", 0.0, "Establish the lower limit of the parameter under study")
	flag.Parse()

	if flag.NArg() == 0 {
		log.Fatalln("Specify a raw file")
	}

	file := flag.Arg(0)

	m, vars, err := ltspice.Raw(file)

	if err != nil {
		log.Fatalln(err.Error())
	}

	if m == nil {
		log.Fatalln("no data matrix found")
	}

	cols := len(m)
	rows := len(m[0])
	n := 1.0

	// Calculate number of runs
	for i := 0; i < rows; i++ {
		// detect LT runs (time == 0)
		if i > 0 && m[0][i] == 0 {
			n++
			i++
		}
	}

	log.Println("runs", n)

	// Correct std.dev for number of samples (c4(n))
	// c4(n) = sqrt( 2 / (n-1) ) * gamma(n/2) / gamma((n-1)/2)
	// See https://en.wikipedia.org/wiki/Unbiased_estimation_of_standard_deviation#Bias_correction
	c4 := 0.0
	if n > 100 {
		c4 = 4 * (n - 1) / (4*n - 3)
	} else {
		c4 = math.Sqrt(2.0/(n-1)) * math.Gamma(n/2) / math.Gamma((n-1)/2)
	}
	log.Println("c4", c4)

	for i := 0; i < cols; i++ {
		p := Parameter{Name: vars[i], Max: math.NaN(), Min: math.NaN()}
		if i > 0 {
			p.Mean = stats.StatsMean(m[i])
			p.MaxHere = stats.StatsMax(m[i])
			p.MinHere = stats.StatsMin(m[i])
			p.StdDev = stats.StatsSampleStandardDeviation(m[i]) / c4 // This includes Bessel correction (which is ok!)
		}
		Parameters = append(Parameters, p)
	}

	// Does the vars list include any _min or _max ?

	for i := 1; i < cols; i++ {

		if strings.HasSuffix(vars[i], "_min)") {
			v := vars[i][0:len(vars[i])-5] + ")"

			j := 1
			for ; j < cols; j++ {
				if vars[j] == v {
					Parameters[j].Min = Parameters[i].Mean
					break
				}
			}
			log.Println("Min", Parameters[j].Name, Parameters[j].Min)
		}

		if strings.HasSuffix(vars[i], "_max)") {
			v := vars[i][0:len(vars[i])-5] + ")"

			j := 1
			for ; j < cols; j++ {
				if vars[j] == v {
					Parameters[j].Max = Parameters[i].Mean
					break
				}
			}
			log.Println("Max", Parameters[j].Name, Parameters[j].Max)
		}
	}

	log.Println("Ntotal", rows)

	// For parameters with min,max calculate additional columns
	for i, p := range Parameters {

		var cpku, cpkl, badl, badu float64

		norm, err := prob.NewNormal(p.Mean, p.StdDev)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		if !math.IsNaN(p.Max) {
			cpku = (p.Max - p.Mean) / (3.0 * p.StdDev)
			badu = 1.0 - norm.Cdf(p.Max)
			Parameters[i].MaxCount = maxcount(m[i], p.Max)
		}

		if !math.IsNaN(p.Min) {
			cpkl = (p.Mean - p.Min) / (3.0 * p.StdDev)
			badl = norm.Cdf(p.Min)
			Parameters[i].MinCount = mincount(m[i], p.Min)
		}

		bad := badu + badl

		Parameters[i].Cpk = math.Min(cpkl, cpku)
		Parameters[i].Good = 1.0 - bad
		Parameters[i].Ppm = bad * 1e6
	}

	if hist > 0 {
		histogram(m[hist], Parameters[hist])
		return
	}

	if rms > 0 {
		Rms(n, m[rms], Parameters[rms], data)
		return
	}

	if duty == 0 {
		if header {
			if csv {
				fmt.Println("parameter, mean, sdev(unbiased), min(found), max(found), min(spec), max(spec), cpk, %ok, ppm, Nmax, Nmin")
			} else {
				fmt.Printf("%-20s %30s %30s %30s %30s %30s %20s %20s %20s %10s %10s %10s\n", "parameter", "mean", "sdev(unbiased)", "min(found)", "max(found)", "min", "max", "cpk", "%ok", "ppm", "Nmax", "Nmin")
			}
		}

		for i, p := range Parameters {

			if i == 0 {
				continue
			}
			if csv {
				fmt.Printf("%s, %g, %g, %g, %g, %g, %g, %g, %.6f, %.1f, %d, %d\n", "'"+p.Name+"'", p.Mean, p.StdDev, p.MinHere, p.MaxHere, p.Min, p.Max, p.Cpk, p.Good*100.0, p.Ppm, p.MaxCount, p.MinCount)
			} else {
				fmt.Printf("%3d %-20s %30g %30g %30g %30g %30g %20g %20g %20.6f %10.1f %10d %10d\n", i, "'"+p.Name+"'", p.Mean, p.StdDev, p.MinHere, p.MaxHere, p.Min, p.Max, p.Cpk, p.Good*100.0, p.Ppm, p.MaxCount, p.MinCount)
			}
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

		//if summary {
		mean := stats.StatsMean(dcs)
		max = stats.StatsMax(dcs)
		min = stats.StatsMin(dcs)
		sdev := stats.StatsSampleStandardDeviation(dcs) / c4
		fmt.Printf("mean, min, max, sdev, cpk, ok, ppm\n")

		cpk := (upperLimit - mean) / (3.0 * sdev)
		norm, err := prob.NewNormal(mean, sdev)
		if err != nil {
			log.Println(err.Error())
		} else {
			bad := (1.0 - norm.Cdf(upperLimit)) * 2
			good := 1.0 - bad
			ppm := bad * 1e6
			fmt.Printf("%f, %f, %f, %f, %f, %f, %f\n", mean, lowerLimit, upperLimit, sdev, cpk, good*100.0, ppm)
		}

	}
}

func Rms(runs float64, vv []float64, p Parameter, data bool) {

	N := len(vv)
	nr := int(runs)
	ns := N / nr

	if !data {
		fmt.Println(p.Name)
		fmt.Println(" - runs: ", nr)
		fmt.Println(" - samples: ", N)
		fmt.Println(" - samples/run: ", ns)
	}

	var r float64
	var rms []float64
	var k int

	for i := 0; i < nr; i++ {
		r = 0
		for j := 0; j < ns; j++ {
			r += vv[k] * vv[k]
			k++
			if data && i == 0 {
				fmt.Println(vv[k])
			}
		}
		r = math.Sqrt(r / float64(ns))
		rms = append(rms, r)
		if data {
			// fmt.Println(r, ns, k)
		}
	}

	if !data {
		mean := stats.StatsMean(rms)
		sd := stats.StatsSampleStandardDeviation(rms)
		max := stats.StatsMax(rms)
		min := stats.StatsMin(rms)

		tolpos := (max - mean) / mean
		tolneg := (mean - min) / mean
		tol := (max - min) / mean / 2 * 100

		fmt.Printf("mean %f, sd %f, min %f, max %f tol +%f-%f tol %f%%\n", mean, sd, min, max, tolpos, tolneg, tol)
	}
}

func maxcount(m []float64, max float64) int {

	n := 0
	for i := 0; i < len(m); i++ {
		if m[i] > max {
			n++
		}
	}
	return n
}

func mincount(m []float64, min float64) int {

	n := 0
	for i := 0; i < len(m); i++ {
		if m[i] < min {
			n++
		}
	}
	return n
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

	// fmt.Printf("%f, %f\n", mean, std)

	return mean, std
}

type V struct {
	vs []float64
}

func (v V) Len() int {
	return len(v.vs)
}

func (v V) Value(i int) float64 {
	return v.vs[i]
}

func histogram(v []float64, p Parameter) {

	// n is the number of bars
	n := 50.0

	// width of bars
	w := 500 / int(n)

	min := stats.StatsMin(v)
	max := stats.StatsMax(v)

	if !math.IsNaN(p.Max) {
		if p.Max > max {
			max = p.Max
		}
		if p.Min < min {
			min = p.Min
		}
	}

	h := make([]float64, int(n)+1)
	step := (max - min) / n

	// log.Printf("%f bins, min %f, max %f, step %f\n", n, min, max, step)

	for i := 0; i < len(v); i++ {
		e := (v[i] - min) / step
		// log.Println(i, v[i], e)
		h[int(e)]++
	}

	hmax := stats.StatsMax(h)
	for i := 0; i < len(h); i++ {
		h[i] /= hmax

	}

	for i := 0; i < int(n); i++ {
		//fmt.Println(h[i])
	}

	fmt.Println(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" width="520" height="510" viewBox="0,0,520,510"><desc>R SVG Plot!</desc><rect width="100%" height="100%" style="fill:#FFFFFF"/>`)
	y0 := 490
	x1 := 10
	for i := 0; i < int(n); i++ {
		y1 := int(490 - h[i]*480)
		fmt.Printf("<polygon points='%d,%d %d,%d %d,%d, %d,%d' style='stroke-width:1;stroke:#999;fill:#ADD8E6;stroke-opacity:1.000000;fill-opacity:1.000000' />\n", x1, y0, x1, y1, x1+w, y1, x1+w, y0)
		x1 += w
	}
	fmt.Printf("<text x='250' y='505' text-anchor='middle' alignment-baseline='bottom' style='font-size:14; font-family: arial'>min=%f, max=%f</text>", min, max)
	//fmt.Printf("<text x='490' y='500' text-anchor='left' alignment-baseline='bottom' style='font-size:12; font-family: arial'>%f</text>", max)
	fmt.Println("</svg>")
}
