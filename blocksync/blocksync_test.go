// Copyright (c) 2019 IoTeX Foundation
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package blocksync

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-proto/golang/iotexrpc"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iotexproject/iotex-core/action/protocol"
	"github.com/iotexproject/iotex-core/action/protocol/account"
	accountutil "github.com/iotexproject/iotex-core/action/protocol/account/util"
	"github.com/iotexproject/iotex-core/action/protocol/rewarding"
	"github.com/iotexproject/iotex-core/action/protocol/rolldpos"
	"github.com/iotexproject/iotex-core/actpool"
	bc "github.com/iotexproject/iotex-core/blockchain"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/config"
	"github.com/iotexproject/iotex-core/db"
	"github.com/iotexproject/iotex-core/state/factory"
	"github.com/iotexproject/iotex-core/test/identityset"
	"github.com/iotexproject/iotex-core/test/mock/mock_blockchain"
	"github.com/iotexproject/iotex-core/test/mock/mock_blockdao"
	"github.com/iotexproject/iotex-core/test/mock/mock_blocksync"
	"github.com/iotexproject/iotex-core/test/mock/mock_consensus"
	"github.com/iotexproject/iotex-core/test/mock/mock_factory"
	"github.com/iotexproject/iotex-core/testutil"
)

var opts = []Option{
	WithUnicastOutBound(func(_ context.Context, _ peerstore.PeerInfo, msg proto.Message) error {
		return nil
	}),
	WithNeighbors(func(_ context.Context) ([]peerstore.PeerInfo, error) { return nil, nil }),
}

func TestNewBlockSyncer(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mBc := mock_blockchain.NewMockBlockchain(ctrl)
	mSf := mock_factory.NewMockFactory(ctrl)
	// TipHeight return ERROR
	mBc.EXPECT().TipHeight().AnyTimes().Return(uint64(0))
	mBc.EXPECT().ChainID().AnyTimes().Return(config.Default.Chain.ID)
	blk := block.NewBlockDeprecated(
		uint32(123),
		uint64(0),
		hash.Hash256{},
		testutil.TimestampNow(),
		identityset.PrivateKey(27).PublicKey(),
		nil,
	)
	dao := mock_blockdao.NewMockBlockDAO(ctrl)
	dao.EXPECT().GetBlockByHeight(gomock.Any()).AnyTimes().Return(blk, nil)

	cfg, err := newTestConfig()
	require.NoError(err)
	ap, err := actpool.NewActPool(mSf, cfg.ActPool, actpool.EnableExperimentalActions())
	assert.NoError(err)

	cs := mock_consensus.NewMockConsensus(ctrl)

	bs, err := NewBlockSyncer(cfg, mBc, dao, ap, cs, opts...)
	assert.Nil(err)
	assert.NotNil(bs)
}

func TestBlockSyncerStart(t *testing.T) {
	assert := assert.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mBs := mock_blocksync.NewMockBlockSync(ctrl)
	mBs.EXPECT().Start(gomock.Any()).Times(1)
	assert.Nil(mBs.Start(ctx))
}

func TestBlockSyncerStop(t *testing.T) {
	assert := assert.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mBs := mock_blocksync.NewMockBlockSync(ctrl)
	mBs.EXPECT().Stop(gomock.Any()).Times(1)
	assert.Nil(mBs.Stop(ctx))
}

func TestBlockSyncerProcessSyncRequest(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mBc := mock_blockchain.NewMockBlockchain(ctrl)
	mSf := mock_factory.NewMockFactory(ctrl)
	mBc.EXPECT().ChainID().AnyTimes().Return(config.Default.Chain.ID)
	blk := block.NewBlockDeprecated(
		uint32(123),
		uint64(0),
		hash.Hash256{},
		testutil.TimestampNow(),
		identityset.PrivateKey(27).PublicKey(),
		nil,
	)
	dao := mock_blockdao.NewMockBlockDAO(ctrl)
	dao.EXPECT().GetBlockByHeight(gomock.Any()).AnyTimes().Return(blk, nil)
	mBc.EXPECT().TipHeight().AnyTimes().Return(uint64(0))
	cfg, err := newTestConfig()
	require.NoError(err)
	ap, err := actpool.NewActPool(mSf, cfg.ActPool, actpool.EnableExperimentalActions())
	assert.NoError(err)
	cs := mock_consensus.NewMockConsensus(ctrl)

	bs, err := NewBlockSyncer(cfg, mBc, dao, ap, cs, opts...)
	assert.NoError(err)

	pbBs := &iotexrpc.BlockSync{
		Start: 1,
		End:   1,
	}
	assert.NoError(bs.ProcessSyncRequest(context.Background(), peerstore.PeerInfo{}, pbBs))
}

func TestBlockSyncerProcessSyncRequestError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg, err := newTestConfig()
	require.NoError(err)

	chain := mock_blockchain.NewMockBlockchain(ctrl)
	sf := mock_factory.NewMockFactory(ctrl)
	dao := mock_blockdao.NewMockBlockDAO(ctrl)
	dao.EXPECT().GetBlockByHeight(uint64(1)).Return(nil, errors.New("some error")).Times(1)
	chain.EXPECT().ChainID().Return(uint32(1)).AnyTimes()
	chain.EXPECT().TipHeight().Return(uint64(10)).Times(1)
	ap, err := actpool.NewActPool(sf, cfg.ActPool, actpool.EnableExperimentalActions())
	require.NotNil(ap)
	require.NoError(err)
	cs := mock_consensus.NewMockConsensus(ctrl)

	bs, err := NewBlockSyncer(cfg, chain, dao, ap, cs, opts...)
	require.NoError(err)
	pbBs := &iotexrpc.BlockSync{
		Start: 1,
		End:   5,
	}
	require.Error(bs.ProcessSyncRequest(context.Background(), peerstore.PeerInfo{}, pbBs))
}

func TestBlockSyncerProcessBlockTipHeight(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	ctx := context.Background()
	cfg, err := newTestConfig()
	require.NoError(err)
	registry := protocol.NewRegistry()
	acc := account.NewProtocol(rewarding.DepositGas)
	require.NoError(acc.Register(registry))
	rp := rolldpos.NewProtocol(cfg.Genesis.NumCandidateDelegates, cfg.Genesis.NumDelegates, cfg.Genesis.NumSubEpochs)
	require.NoError(rp.Register(registry))
	sf, err := factory.NewFactory(cfg, factory.InMemTrieOption())
	require.NoError(err)
	dao := blockdao.NewBlockDAO(db.NewMemKVStore(), nil, cfg.Chain.CompressBlock, cfg.DB)
	chain := bc.NewBlockchain(
		cfg,
		dao,
		sf,
		bc.RegistryOption(registry),
	)
	chain.Validator().AddActionEnvelopeValidators(protocol.NewGenericValidator(sf, accountutil.AccountState))
	require.NoError(chain.Start(ctx))
	require.NotNil(chain)
	ap, err := actpool.NewActPool(sf, cfg.ActPool, actpool.EnableExperimentalActions())
	require.NotNil(ap)
	require.NoError(err)
	cs := mock_consensus.NewMockConsensus(ctrl)
	cs.EXPECT().ValidateBlockFooter(gomock.Any()).Return(nil).Times(1)
	cs.EXPECT().Calibrate(uint64(1)).Times(1)

	bs, err := NewBlockSyncer(cfg, chain, dao, ap, cs, opts...)
	require.NoError(err)

	defer func() {
		require.NoError(chain.Stop(ctx))
		ctrl.Finish()
	}()

	h := chain.TipHeight()
	blk, err := chain.MintNewBlock(
		nil,
		testutil.TimestampNow(),
	)
	require.NotNil(blk)
	require.NoError(err)
	require.NoError(bs.ProcessBlock(ctx, blk.Block))
	h2 := chain.TipHeight()
	assert.Equal(t, h+1, h2)

	// commit top
	require.NoError(bs.ProcessBlock(ctx, blk.Block))
	h3 := chain.TipHeight()
	assert.Equal(t, h+1, h3)

	// commit same block again
	require.NoError(bs.ProcessBlock(ctx, blk.Block))
	h4 := chain.TipHeight()
	assert.Equal(t, h3, h4)
}

func TestBlockSyncerProcessBlockOutOfOrder(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	ctx := context.Background()
	cfg, err := newTestConfig()
	require.NoError(err)
	registry := protocol.NewRegistry()
	acc := account.NewProtocol(rewarding.DepositGas)
	require.NoError(acc.Register(registry))
	rp := rolldpos.NewProtocol(cfg.Genesis.NumCandidateDelegates, cfg.Genesis.NumDelegates, cfg.Genesis.NumSubEpochs)
	require.NoError(rp.Register(registry))
	sf, err := factory.NewFactory(cfg, factory.InMemTrieOption())
	require.NoError(err)
	dao := blockdao.NewBlockDAO(db.NewMemKVStore(), nil, cfg.Chain.CompressBlock, cfg.DB)
	chain1 := bc.NewBlockchain(
		cfg,
		dao,
		sf,
		bc.RegistryOption(registry),
	)
	require.NotNil(chain1)
	chain1.Validator().AddActionEnvelopeValidators(protocol.NewGenericValidator(sf, accountutil.AccountState))
	require.NoError(chain1.Start(ctx))
	ap1, err := actpool.NewActPool(sf, cfg.ActPool, actpool.EnableExperimentalActions())
	require.NotNil(ap1)
	require.NoError(err)
	cs1 := mock_consensus.NewMockConsensus(ctrl)
	cs1.EXPECT().ValidateBlockFooter(gomock.Any()).Return(nil).Times(3)
	cs1.EXPECT().Calibrate(gomock.Any()).Times(3)

	bs1, err := NewBlockSyncer(cfg, chain1, dao, ap1, cs1, opts...)
	require.NoError(err)
	registry2 := protocol.NewRegistry()
	require.NoError(acc.Register(registry2))
	require.NoError(rp.Register(registry2))
	sf2, err := factory.NewFactory(cfg, factory.InMemTrieOption())
	require.NoError(err)
	dao2 := blockdao.NewBlockDAO(db.NewMemKVStore(), nil, cfg.Chain.CompressBlock, cfg.DB)
	chain2 := bc.NewBlockchain(
		cfg,
		dao2,
		sf2,
		bc.RegistryOption(registry2),
	)
	require.NotNil(chain2)
	chain2.Validator().AddActionEnvelopeValidators(protocol.NewGenericValidator(sf2, accountutil.AccountState))
	require.NoError(chain2.Start(ctx))
	ap2, err := actpool.NewActPool(sf2, cfg.ActPool, actpool.EnableExperimentalActions())
	require.NotNil(ap2)
	require.NoError(err)
	cs2 := mock_consensus.NewMockConsensus(ctrl)
	cs2.EXPECT().ValidateBlockFooter(gomock.Any()).Return(nil).Times(3)
	cs2.EXPECT().Calibrate(gomock.Any()).Times(3)
	bs2, err := NewBlockSyncer(cfg, chain2, dao2, ap2, cs2, opts...)
	require.NoError(err)

	defer func() {
		require.NoError(chain1.Stop(ctx))
		require.NoError(chain2.Stop(ctx))
		ctrl.Finish()
	}()

	// commit top
	blk1, err := chain1.MintNewBlock(
		nil,
		testutil.TimestampNow(),
	)
	require.NotNil(blk1)
	require.NoError(err)
	require.NoError(bs1.ProcessBlock(ctx, blk1.Block))
	blk2, err := chain1.MintNewBlock(
		nil,
		testutil.TimestampNow(),
	)
	require.NotNil(blk2)
	require.NoError(err)
	require.NoError(bs1.ProcessBlock(ctx, blk2.Block))
	blk3, err := chain1.MintNewBlock(
		nil,
		testutil.TimestampNow(),
	)
	require.NotNil(blk3)
	require.NoError(err)
	require.NoError(bs1.ProcessBlock(ctx, blk3.Block))
	h1 := chain1.TipHeight()
	assert.Equal(t, uint64(3), h1)

	require.NoError(bs2.ProcessBlock(ctx, blk3.Block))
	require.NoError(bs2.ProcessBlock(ctx, blk2.Block))
	require.NoError(bs2.ProcessBlock(ctx, blk2.Block))
	require.NoError(bs2.ProcessBlock(ctx, blk1.Block))
	h2 := chain2.TipHeight()
	assert.Equal(t, h1, h2)
}

func TestBlockSyncerProcessBlockSync(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	ctx := context.Background()
	cfg, err := newTestConfig()
	require.NoError(err)
	registry := protocol.NewRegistry()
	acc := account.NewProtocol(rewarding.DepositGas)
	require.NoError(acc.Register(registry))
	rolldposProtocol := rolldpos.NewProtocol(
		cfg.Genesis.NumCandidateDelegates,
		cfg.Genesis.NumDelegates,
		cfg.Genesis.NumSubEpochs,
	)
	require.NoError(rolldposProtocol.Register(registry))
	sf, err := factory.NewFactory(cfg, factory.InMemTrieOption())
	require.NoError(err)
	dao := blockdao.NewBlockDAO(db.NewMemKVStore(), nil, cfg.Chain.CompressBlock, cfg.DB)
	chain1 := bc.NewBlockchain(
		cfg,
		dao,
		sf,
		bc.RegistryOption(registry),
	)
	chain1.Validator().AddActionEnvelopeValidators(protocol.NewGenericValidator(sf, accountutil.AccountState))
	require.NoError(chain1.Start(ctx))
	require.NotNil(chain1)
	ap1, err := actpool.NewActPool(sf, cfg.ActPool)
	require.NotNil(ap1)
	require.NoError(err)
	cs1 := mock_consensus.NewMockConsensus(ctrl)
	cs1.EXPECT().ValidateBlockFooter(gomock.Any()).Return(nil).Times(3)
	cs1.EXPECT().Calibrate(gomock.Any()).Times(3)
	bs1, err := NewBlockSyncer(cfg, chain1, dao, ap1, cs1, opts...)
	require.NoError(err)
	registry2 := protocol.NewRegistry()
	require.NoError(acc.Register(registry2))
	require.NoError(rolldposProtocol.Register(registry2))
	sf2, err := factory.NewFactory(cfg, factory.InMemTrieOption())
	require.NoError(err)
	dao2 := blockdao.NewBlockDAO(db.NewMemKVStore(), nil, cfg.Chain.CompressBlock, cfg.DB)
	chain2 := bc.NewBlockchain(
		cfg,
		dao2,
		sf2,
		bc.RegistryOption(registry2),
	)
	chain2.Validator().AddActionEnvelopeValidators(protocol.NewGenericValidator(sf2, accountutil.AccountState))
	require.NoError(chain2.Start(ctx))
	require.NotNil(chain2)
	ap2, err := actpool.NewActPool(sf2, cfg.ActPool, actpool.EnableExperimentalActions())
	require.NotNil(ap2)
	require.NoError(err)
	cs2 := mock_consensus.NewMockConsensus(ctrl)
	cs2.EXPECT().ValidateBlockFooter(gomock.Any()).Return(nil).Times(3)
	cs2.EXPECT().Calibrate(gomock.Any()).Times(3)
	bs2, err := NewBlockSyncer(cfg, chain2, dao2, ap2, cs2, opts...)
	require.NoError(err)

	defer func() {
		require.NoError(chain1.Stop(ctx))
		require.NoError(chain2.Stop(ctx))
		ctrl.Finish()
	}()

	// commit top
	blk1, err := chain1.MintNewBlock(
		nil,
		testutil.TimestampNow(),
	)
	require.NotNil(blk1)
	require.NoError(err)
	require.NoError(bs1.ProcessBlock(ctx, blk1.Block))
	blk2, err := chain1.MintNewBlock(
		nil,
		testutil.TimestampNow(),
	)
	require.NotNil(blk2)
	require.NoError(err)
	require.NoError(bs1.ProcessBlock(ctx, blk2.Block))
	blk3, err := chain1.MintNewBlock(
		nil,
		testutil.TimestampNow(),
	)
	require.NotNil(blk3)
	require.NoError(err)
	require.NoError(bs1.ProcessBlock(ctx, blk3.Block))
	h1 := chain1.TipHeight()
	assert.Equal(t, uint64(3), h1)

	require.NoError(bs2.ProcessBlockSync(ctx, blk3.Block))
	require.NoError(bs2.ProcessBlockSync(ctx, blk2.Block))
	require.NoError(bs2.ProcessBlockSync(ctx, blk1.Block))
	h2 := chain2.TipHeight()
	assert.Equal(t, h1, h2)
}

func TestBlockSyncerSync(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	ctx := context.Background()
	cfg, err := newTestConfig()
	require.NoError(err)
	registry := protocol.NewRegistry()
	rp := rolldpos.NewProtocol(cfg.Genesis.NumCandidateDelegates, cfg.Genesis.NumDelegates, cfg.Genesis.NumSubEpochs)
	require.NoError(rp.Register(registry))
	sf, err := factory.NewFactory(cfg, factory.InMemTrieOption())
	require.NoError(err)
	dao := blockdao.NewBlockDAO(db.NewMemKVStore(), nil, cfg.Chain.CompressBlock, cfg.DB)
	chain := bc.NewBlockchain(cfg, dao, sf, bc.RegistryOption(registry))
	require.NoError(chain.Start(ctx))
	require.NotNil(chain)
	ap, err := actpool.NewActPool(sf, cfg.ActPool, actpool.EnableExperimentalActions())
	require.NotNil(ap)
	require.NoError(err)
	cs := mock_consensus.NewMockConsensus(ctrl)
	cs.EXPECT().ValidateBlockFooter(gomock.Any()).Return(nil).Times(2)
	cs.EXPECT().Calibrate(gomock.Any()).Times(2)

	bs, err := NewBlockSyncer(cfg, chain, dao, ap, cs, opts...)
	require.NotNil(bs)
	require.NoError(err)
	require.NoError(bs.Start(ctx))
	time.Sleep(time.Millisecond << 7)

	defer func() {
		require.NoError(bs.Stop(ctx))
		require.NoError(chain.Stop(ctx))
		ctrl.Finish()
	}()

	blk, err := chain.MintNewBlock(
		nil,
		testutil.TimestampNow(),
	)
	require.NotNil(blk)
	require.NoError(err)
	require.NoError(bs.ProcessBlock(ctx, blk.Block))

	blk, err = chain.MintNewBlock(
		nil,
		testutil.TimestampNow(),
	)
	require.NotNil(blk)
	require.NoError(err)
	require.NoError(bs.ProcessBlock(ctx, blk.Block))
	time.Sleep(time.Millisecond << 7)
}

func newTestConfig() (config.Config, error) {
	testTrieFile, err := ioutil.TempFile(os.TempDir(), "trie")
	if err != nil {
		return config.Config{}, err
	}
	testTriePath := testTrieFile.Name()
	testDBFile, err := ioutil.TempFile(os.TempDir(), "db")
	if err != nil {
		return config.Config{}, err
	}
	testDBPath := testDBFile.Name()

	cfg := config.Default
	cfg.Chain.TrieDBPath = testTriePath
	cfg.Chain.ChainDBPath = testDBPath
	cfg.BlockSync.Interval = 100 * time.Millisecond
	cfg.Consensus.Scheme = config.NOOPScheme
	cfg.Network.Host = "127.0.0.1"
	cfg.Network.Port = 10000
	cfg.Network.BootstrapNodes = []string{"127.0.0.1:10000", "127.0.0.1:4689"}
	cfg.Genesis.EnableGravityChainVoting = false
	return cfg, nil
}
