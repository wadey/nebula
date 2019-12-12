package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/slackhq/nebula/cert"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

type printFlags struct {
	set  *flag.FlagSet
	json *bool
	path *string
}

func newPrintFlags() *printFlags {
	pf := printFlags{set: flag.NewFlagSet("print", flag.ContinueOnError)}
	pf.set.Usage = func() {}
	pf.json = pf.set.Bool("json", false, "Optional: outputs certificates in json format")
	pf.path = pf.set.String("path", "", "Required: path to the certificate")

	return &pf
}

func printCert(args []string, out io.Writer, errOut io.Writer) error {
	pf := newPrintFlags()
	err := pf.set.Parse(args)
	if err != nil {
		return err
	}

	if err := mustFlagString("path", pf.path); err != nil {
		return err
	}

	rawCert, err := ioutil.ReadFile(*pf.path)
	if err != nil {
		return fmt.Errorf("unable to read cert; %s", err)
	}

	var c *cert.NebulaCertificate

	for {
		c, rawCert, err = cert.UnmarshalNebulaCertificateFromPEM(rawCert)
		if err != nil {
			return fmt.Errorf("error while unmarshaling cert: %s", err)
		}

		if *pf.json {
			b, _ := json.Marshal(c)
			out.Write(b)
			out.Write([]byte("\n"))

		} else {
			out.Write([]byte(c.String()))
			out.Write([]byte("\n"))
		}

		if rawCert == nil || len(rawCert) == 0 || strings.TrimSpace(string(rawCert)) == "" {
			break
		}
	}

	return nil
}

func printSummary() string {
	return "print <flags>: prints details about a certificate"
}

func printHelp(out io.Writer) {
	pf := newPrintFlags()
	out.Write([]byte("Usage of " + os.Args[0] + " " + printSummary() + "\n"))
	pf.set.SetOutput(out)
	pf.set.PrintDefaults()
}
