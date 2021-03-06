// Copyright (c) 2019 IoTeX Foundation
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package action

import (
	"math/big"

	"github.com/spf13/cobra"

	"github.com/iotexproject/iotex-core/ioctl/output"
	"github.com/iotexproject/iotex-core/ioctl/util"
)

// actionDeployCmd represents the action deploy command
var actionDeployCmd = &cobra.Command{
	Use:   "deploy [AMOUNT_IOTX] [-s SIGNER] -b BYTE_CODE [-l GAS_LIMIT] [-p GAS_PRICE] [-P PASSWORD] [-y]",
	Short: "Deploy smart contract on IoTeX blockchain",
	Args:  cobra.MaximumNArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		bytecode, err := decodeBytecode()
		if err != nil {
			return output.PrintError(output.FlagError, "Invalid bytecode flag:"+err.Error())
		}
		amount := big.NewInt(0)
		if len(args) == 1 {
			amount, err = util.StringToRau(args[0], util.IotxDecimalNum)
			if err != nil {
				return output.PrintError(output.ConvertError, "Invalid amount:"+err.Error())
			}
		}
		return execute("", amount, bytecode)
	},
}

func init() {
	registerWriteCommand(actionDeployCmd)
	bytecodeFlag.RegisterCommand(actionDeployCmd)
	bytecodeFlag.MarkFlagRequired(actionDeployCmd)
}
