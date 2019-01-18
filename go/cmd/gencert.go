// Copyright (C) 2018 Cranky Kernel
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/crankykernel/maker/go/gencert"
)

var opts gencert.Flags

var gencertCmd = &cobra.Command{
	Use:   "gencert",
	Short: "Generate a self signed TLS certificate/key pair.",
	Run: func(cmd *cobra.Command, args []string) {
		gencert.GenCertMain(opts, args)
	},
}

func init() {
	rootCmd.AddCommand(gencertCmd)

	opts.Host = gencertCmd.Flags().String("host", gencert.DEFAULT_HOST, "Hostnames and/or IPs to generate cert for.")
	opts.Org = gencertCmd.Flags().String("org", gencert.DEFAULT_ORG, "Organization name for certificate.")
	opts.Filename = gencertCmd.Flags().String("filename", gencert.DEFAULT_FILENAME, "Output filename")
}
