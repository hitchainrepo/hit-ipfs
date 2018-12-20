package main

import (
	"fmt"
	"github.com/sparrc/go-ping"
	"time"
)

func main() {
	//p := fastping.NewPinger()
	//ra, err := net.ResolveIPAddr("ip4:icmp", "47.89.185.83")
	//if err != nil {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}
	//p.AddIPAddr(ra)
	//p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
	//	fmt.Printf("IP Addr: %s receive, RTT: %v\n", addr.String(), rtt)
	//}
	//
	//p.OnIdle = func() {
	//	fmt.Println("finish")
	//}
	//err = p.Run()
	//if err != nil {
	//	fmt.Println(err)
	//}

	//delay := -1.0
	pinger, err := ping.NewPinger("127.0.0.1")
	pinger.Timeout = time.Second * 5 // timeout in 5 seconds
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	}

	pinger.OnRecv = func(pkt *ping.Packet) {
		//fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v\n",
		//	pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt)
		fmt.Printf("%s, delay: %v\n", pkt.IPAddr, pkt.Rtt)
		delay := pkt.Rtt.Seconds() * 1000 // milliseconds
		fmt.Println(delay)
		pinger.Stop()
	}
	pinger.OnFinish = func(stats *ping.Statistics) {
		fmt.Printf("\n--- %s ping statistics ---\n", stats.Addr)
		fmt.Printf("%d packets transmitted, %d packets received, %v%% packet loss\n",
			stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
		fmt.Printf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
			stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
	}

	fmt.Printf("PING %s (%s):\n", pinger.Addr(), pinger.IPAddr())
	pinger.Run()

	//pinger, err := ping.NewPinger("47.89.185.83")
	//if err != nil {
	//	panic(err)
	//}
	//
	//pinger.Count = 1
	//pinger.Run() // blocks until finished
	//stats := pinger.Statistics() // get send/receive/rtt stats
	//fmt.Print(stats)
}