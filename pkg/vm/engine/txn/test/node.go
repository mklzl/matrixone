// Copyright 2022 Matrix Origin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testtxnengine

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	logservicepb "github.com/matrixorigin/matrixone/pkg/pb/logservice"
	"github.com/matrixorigin/matrixone/pkg/pb/metadata"
	"github.com/matrixorigin/matrixone/pkg/testutil"
	"github.com/matrixorigin/matrixone/pkg/txn/service"
	txnstorage "github.com/matrixorigin/matrixone/pkg/txn/storage/txn"
	"go.uber.org/zap"
)

type Node struct {
	info logservicepb.DNStore
	// one node, one shard, one service
	service service.TxnService
	shard   metadata.DNShard
}

func (t *testEnv) NewNode(id uint64) *Node {

	shard := metadata.DNShard{
		DNShardRecord: metadata.DNShardRecord{
			ShardID:    id,
			LogShardID: id,
		},
		ReplicaID: id,
		Address:   fmt.Sprintf("shard-%d", id),
	}

	storage, err := txnstorage.New(
		txnstorage.NewMemHandler(testutil.NewMheap(), txnstorage.IsolationPolicy{
			Read: txnstorage.ReadCommitted,
		}),
	)
	if err != nil {
		panic(err)
	}

	nodeInfo := logservicepb.DNStore{
		UUID:           uuid.NewString(),
		ServiceAddress: shard.Address,
		State:          logservicepb.NormalState,
		Shards: []logservicepb.DNShardInfo{
			{
				ShardID:   id,
				ReplicaID: id,
			},
		},
	}

	loggerConfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:      true,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	logger, err := loggerConfig.Build()
	if err != nil {
		panic(err)
	}

	service := service.NewTxnService(
		logger,
		shard,
		storage,
		t.sender,
		t.clock,
		time.Second*61,
	)
	if err := service.Start(); err != nil {
		panic(err)
	}

	node := &Node{
		info:    nodeInfo,
		service: service,
		shard:   shard,
	}

	return node
}
