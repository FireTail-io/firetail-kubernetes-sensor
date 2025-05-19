package main

import (
	"C"
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)
import "time"

func findLibSSL() (string, error) {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("find /usr/lib -name 'libssl.so*'"))
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run ldconfig: %v", err)
	}
	scanner := bufio.NewScanner(&out)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	return "", fmt.Errorf("library not found")
}

func main() {
	// Find the libssl library
	log.Println("Finding libssl...")
	libSSLPath, err := findLibSSL()
	if err != nil {
		log.Fatalf("Error finding libssl: %v", err)
	}
	log.Printf("Found libssl at: %s", libSSLPath)

	// open the libssl executable
	libSSLExecutable, err := link.OpenExecutable(libSSLPath)
	if err != nil {
		log.Fatalf("Err opening libssl executable: %s", err)
	}

	// load the eBPF program
	collection, err := ebpf.LoadCollectionSpec("ssl_read.o")
	if err != nil {
		log.Fatalf("Err loading ssl_read.o: %s", err)
	}
	programs := struct {
		ProbeSslRead  *ebpf.Program `ebpf:"probe_ssl_read"`
		ProbeSslWrite *ebpf.Program `ebpf:"probe_ssl_write"`
	}{}
	err = collection.LoadAndAssign(&programs, nil)
	if err != nil {
		log.Fatalf("Err loading and assigning: %s", err)
	}
	defer programs.ProbeSslRead.Close()

	// attach to the "SSL_read" symbol
	sslReadLink, err := libSSLExecutable.Uprobe("SSL_read", programs.ProbeSslRead, nil)
	if err != nil {
		log.Fatalf("Err attaching uprobe: %v", err)
	}
	defer sslReadLink.Close()

	// attach to the "SSL_read" symbol
	sslWriteLink, err := libSSLExecutable.Uprobe("SSL_write", programs.ProbeSslWrite, nil)
	if err != nil {
		log.Fatalf("Err attaching uprobe: %v", err)
	}
	defer sslWriteLink.Close()

	t := time.NewTicker(time.Second)
	for {
		select {
		case <-t.C:
			log.Println("Running...")
		}
	}
}
