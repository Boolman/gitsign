//
// Copyright 2022 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/pborman/getopt/v2"
	"github.com/pkg/errors"

	// Enable OIDC providers
	_ "github.com/sigstore/cosign/pkg/providers/all"
)

const (
	// TODO: Use fulcio as timestamp authority.
	defaultTSA = ""
)

var (

	// Action flags
	helpFlag    = getopt.BoolLong("help", 'h', "print this help message")
	versionFlag = getopt.BoolLong("version", 'v', "print the version number")
	signFlag    = getopt.BoolLong("sign", 's', "make a signature")
	verifyFlag  = getopt.BoolLong("verify", 0, "verify a signature")

	// Option flags
	localUserOpt    = getopt.StringLong("local-user", 'u', "", "use USER-ID to sign", "USER-ID")
	detachSignFlag  = getopt.BoolLong("detach-sign", 'b', "make a detached signature")
	armorFlag       = getopt.BoolLong("armor", 'a', "create ascii armored output")
	statusFdOpt     = getopt.IntLong("status-fd", 0, -1, "write special status strings to the file descriptor n.", "n")
	tsaOpt          = getopt.StringLong("timestamp-authority", 't', defaultTSA, "URL of RFC3161 timestamp authority to use for timestamping", "url")
	includeCertsOpt = getopt.IntLong("include-certs", 0, -2, "-3 is the same as -2, but ommits issuer when cert has Authority Information Access extension. -2 includes all certs except root. -1 includes all certs. 0 includes no certs. 1 includes leaf cert. >1 includes n from the leaf. Default -2.", "n")

	// Remaining arguments
	fileArgs []string

	// these are changed in tests
	stdin  io.ReadCloser  = os.Stdin
	stdout io.WriteCloser = os.Stdout
	stderr io.Writer      = os.Stderr
)

func main() {
	if err := runCommand(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCommand() error {
	// Parse CLI args
	getopt.HelpColumn = 40
	getopt.SetParameters("[files]")
	getopt.Parse()
	fileArgs = getopt.Args()

	if logPath := os.Getenv("GITSIGN_LOG"); logPath != "" {
		// Since Git eats both stdout and stderr, we don't have a good way of
		// getting error information back from clients if things go wrong.
		// As a janky way to preserve error message, tee stderr to
		// a temp file.
		if f, err := os.Create(logPath); err == nil {
			defer f.Close()
			stderr = io.MultiWriter(stderr, f)
		}
	}

	if *helpFlag {
		getopt.Usage()
		return nil
	}

	if *versionFlag {
		version := "unknown"
		info, ok := debug.ReadBuildInfo()
		if ok {
			for _, s := range info.Settings {
				if s.Key == "vcs.revision" {
					version = s.Value
				}
			}
		}
		fmt.Println(version)
		return nil
	}

	if *signFlag {
		if *verifyFlag {
			return errors.New("specify --help, --sign, or --verify")
		}
		if len(*localUserOpt) == 0 {
			return errors.New("specify a USER-ID to sign with")
		}

		return commandSign()
	}

	if *verifyFlag {
		if *signFlag {
			return errors.New("specify --help, --sign, or --verify")
		}
		if len(*localUserOpt) > 0 {
			return errors.New("local-user cannot be specified for verification")
		}
		if *detachSignFlag {
			return errors.New("detach-sign cannot be specified for verification")
		}
		if *armorFlag {
			return errors.New("armor cannot be specified for verification")
		}

		return commandVerify()
	}

	return errors.New("specify --help, --sign, --verify, or --list-keys")
}
