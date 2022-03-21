package main

import (
	"flag"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type TTD struct {
	*big.Int
}

func (t *TTD) Set(val string) error {
	dec := big.NewInt(0)
	if _, suc := dec.SetString(val, 0); !suc {
		return fmt.Errorf("Unable to parse %s", val)
	}
	*t = TTD{dec}
	return nil
}

type Prober struct {
	ELClients ELClients
	CLClients CLClients
	Probes    VerificationProbes
	WaitGroup sync.WaitGroup
	StopChan  chan struct{}
}

func (p *Prober) StartProbes() {
	p.StopChan = make(chan struct{})
	for _, vp := range p.Probes {
		vp := vp
		p.WaitGroup.Add(1)
		go vp.Loop(p.StopChan, p.WaitGroup)
	}
}

func (p *Prober) StopProbes() {
	close(p.StopChan)
}

func (p *Prober) WrapUp() {
	for _, vp := range p.Probes {

		if vOut, err := vp.Verify(); err != nil {
			fmt.Printf("ERR(%s): %s\n", vp.Verification.VerificationName, err)
		} else {
			fmt.Printf("%s\n", vOut.String(vp.Verification.VerificationName))
		}
	}
}

func main() {
	var (
		elClients ELClients
		clClients CLClients
		ttd       TTD
	)
	flag.Var(&elClients, "exec-client",
		"Execution client RPC endpoint to check for the client's status")
	flag.Var(&clClients, "beacon-client",
		"Consensus client REST API endpoint to check for the client's status")
	flag.Var(&ttd, "ttd",
		"Value of the Terminal Total Difficulty for the subscribed clients")
	flag.Parse()

	prober := Prober{
		ELClients: elClients,
		CLClients: clClients,
		Probes:    make([]VerificationProbe, 0),
	}

	updateAllTTDTimestamps := func(timestamp uint64) {
		for _, cl := range clClients {
			cl.UpdateTTDTimestamp(timestamp)
		}
	}

	for _, el := range elClients {
		el.TTD = ttd
		el.UpdateTTDTimestamp = updateAllTTDTimestamps
		prober.Probes = append(prober.Probes, NewVerificationProbes(el, AllVerifications)...)
	}

	for _, cl := range clClients {
		cl.TTD = ttd
		prober.Probes = append(prober.Probes, NewVerificationProbes(cl, AllVerifications)...)
	}

	if prober.Probes.ExecutionVerifications() == 0 {
		fmt.Printf("At least 1 execution layer verification is required")
		os.Exit(1)
	}

	prober.StartProbes()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case <-sigs:
			// Need to wait here for the clients to finish up before continuing
			prober.StopProbes()
			prober.WrapUp()
			return
		}
	}

}
