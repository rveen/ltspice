package main

import (
	"log"
	"ltspice"
	"flag"
	"fmt"
	"github.com/GaryBoone/GoStats/stats"
)

func main() {

    var header bool

	flag.BoolVar(&header, "t", false, "Print text header")
	flag.Parse()
	
	file := ""
	if len(flag.Args())>0 {
	    file = flag.Args()[0]
	}

	m, vars, err := ltspice.Raw(file,header)

	if err != nil {
		log.Println(err)
	}
	
	if m==nil {
	   log.Println("null matrix")
	}
	
	col := len(m)
	row := len(m[0])
	
	if header {
	    for j:=1;j<col;j++ {
	        fmt.Printf("%-20s %30g %30g\n","'"+vars[j]+"'",stats.StatsMean(m[j]),stats.StatsSampleStandardDeviation(m[j]))
	    }
	}
	
	if header {
	    return
	}
	
	for i:=0;i<row;i++ {
	    for j:=0;j<col;j++ {
	        fmt.Printf("%g",m[j][i])
	        if j<col-1 {
	            fmt.Print(", ")
	        }
	    }
	    fmt.Println("")
	}
	
}