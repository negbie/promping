package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	promPort = flag.Int("p", 9876, "Expose Prometheus metrics on this port.")
	targets  = flag.String("t", "localhost", "Single or comma seperated targets")
	interval = flag.Int("i", 1, "Ping interval in seconds")
	dry      = flag.Bool("d", false, "Dry mode prints only to console")

	rttMin = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "promping_rtt_min_milliseconds",
		Help: "Min RTT time"},
		[]string{"host"})

	rttAvg = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "promping_rtt_avg_milliseconds",
		Help: "Avg RTT time"},
		[]string{"host"})

	rttMax = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "promping_rtt_max_milliseconds",
		Help: "Max RTT time"},
		[]string{"host"})

	rttLost = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "promping_rtt_lost_total",
		Help: "Lost ping responses"},
		[]string{"host"})
)

func init() {
	prometheus.MustRegister(rttMin)
	prometheus.MustRegister(rttAvg)
	prometheus.MustRegister(rttMax)
	prometheus.MustRegister(rttLost)
}

func splitSlash(c rune) bool {
	return c == '/'
}

func main() {
	if !isInstalled("/usr/bin/fping") {
		log.Fatal("please install fping under /usr/bin/fping")
	}

	flag.Parse()

	if *interval <= 0 {
		log.Fatal("please increase your ping interval")
	}

	http.Handle("/metrics", promhttp.Handler())
	go func() { log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *promPort), nil)) }()

	Q := strconv.Itoa(*interval)
	p := strconv.Itoa(*interval * 100)

	fpingArgs := []string{"-B 1", "-D", "-r0", "-O 0", "-Q " + Q, "-p " + p, "-l"}
	hosts := strings.Split(*targets, ",")
	for _, host := range hosts {
		fpingArgs = append(fpingArgs, host)
	}

	cmd := exec.Command("/usr/bin/fping", fpingArgs...)
	stdPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	var min, avg, max, lost string
	var fmin, favg, fmax, flost float64
	buff := bufio.NewScanner(stdPipe)

	for buff.Scan() {
		text := buff.Text()
		fields := strings.Fields(text)
		if len(fields) > 1 {
			host := fields[0]
			data := fields[4]
			sep := strings.FieldsFunc(data, splitSlash)
			sep[2] = strings.TrimRight(sep[2], "%,")
			_, _, lost = sep[0], sep[1], sep[2]

			if len(fields) > 5 {
				t := fields[7]
				sp := strings.FieldsFunc(t, splitSlash)
				min, avg, max = sp[0], sp[1], sp[2]
			}
			if !*dry {
				if fmin, err = toFloat(min); err != nil {
					continue
				} else {
					rttMin.WithLabelValues(host).Set(fmin)
				}
				if favg, err = toFloat(avg); err != nil {
					continue
				} else {
					rttAvg.WithLabelValues(host).Set(favg)
				}
				if fmax, err = toFloat(max); err != nil {
					continue
				} else {
					rttMax.WithLabelValues(host).Set(fmax)
				}
				if flost, err = toFloat(lost); err != nil {
					continue
				} else {
					rttLost.WithLabelValues(host).Set(flost)
				}
			} else {
				log.Printf("host: %s, min: %v, avg: %v, max: %v, lost: %v\n", host, min, avg, max, lost)
			}
		}
	}
}

func toFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func isInstalled(cp string) bool {
	cmd := exec.Command(cp, "-v")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
