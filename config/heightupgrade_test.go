// Copyright (c) 2019 IoTeX
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHeightChange(t *testing.T) {
	require := require.New(t)

	require.Equal(0, Pacific)
	require.Equal(1, Aleutian)
	cfg := Default
	cfg.Genesis.PacificBlockHeight = uint64(432001)
	hu := NewHeightUpgrade(cfg)
	require.Equal(uint64(432001), hu.pacificHeight)
	require.Equal(uint64(864001), hu.aleutianHeight)

	require.True(hu.IsPre(uint64(432000), Pacific))
	require.True(hu.IsPost(uint64(432001), Pacific))
	require.True(hu.IsPre(uint64(864000), Aleutian))
	require.True(hu.IsPost(uint64(864001), Aleutian))
}
