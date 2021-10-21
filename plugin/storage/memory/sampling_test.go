// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memory

import (
	"testing"
	"time"

	"github.com/crossdock/crossdock-go/assert"
	"github.com/jaegertracing/jaeger/cmd/collector/app/sampling/model"
)

func withPopulatedSamplingStore(f func(samplingStore *SamplingStore)) {
	now := time.Now()
	millisAfter := now.Add(time.Millisecond * time.Duration(100))
	secondsAfter := now.Add(time.Second * time.Duration(2))
	throughputs := []*memoryThroughput{
		{&model.Throughput{Service: "svc-1", Operation: "op-1", Count: 1}, now},
		{&model.Throughput{Service: "svc-1", Operation: "op-2", Count: 1}, millisAfter},
		{&model.Throughput{Service: "svc-2", Operation: "op-3", Count: 1}, secondsAfter},
	}
	pQPS := []*memoryServiceOperationProbabilitiesAndQPS{
		{hostname: "guntur38ab8928", probabilities: model.ServiceOperationProbabilities{"svc-1": {"op-1": 0.01}}, qps: model.ServiceOperationQPS{"svc-1": {"op-1": 10.0}}, time: now},
		{hostname: "peta0242ac130003", probabilities: model.ServiceOperationProbabilities{"svc-1": {"op-2": 0.008}}, qps: model.ServiceOperationQPS{"svc-1": {"op-2": 4.0}}, time: millisAfter},
		{hostname: "tenali11ec8d3d", probabilities: model.ServiceOperationProbabilities{"svc-2": {"op-3": 0.003}}, qps: model.ServiceOperationQPS{"svc-1": {"op-1": 7.0}}, time: secondsAfter},
	}
	samplingStore := &SamplingStore{throughputs: throughputs, probabilitiesAndQPS: pQPS}
	f(samplingStore)
}

func withMemorySamplingStore(f func(samplingStore *SamplingStore)) {
	f(NewSamplingStore())
}

func TestInsertThroughtput(t *testing.T) {
	withMemorySamplingStore(func(samplingStore *SamplingStore) {
		throughputs := []*model.Throughput{
			{Service: "my-svc", Operation: "op"},
			{Service: "our-svc", Operation: "op2"},
		}
		assert.NoError(t, samplingStore.InsertThroughput(throughputs))
		assert.Equal(t, 2, len(samplingStore.throughputs))
	})
}

func TestGetThroughput(t *testing.T) {
	withPopulatedSamplingStore(func(samplingStore *SamplingStore) {
		start := time.Now()
		ret, err := samplingStore.GetThroughput(start, start.Add(time.Second*time.Duration(1)))
		assert.NoError(t, err)
		assert.Equal(t, 1, len(ret))
		ret1, _ := samplingStore.GetThroughput(start, start)
		assert.Equal(t, 0, len(ret1))
		ret2, _ := samplingStore.GetThroughput(start, start.Add(time.Hour*time.Duration(1)))
		assert.Equal(t, 2, len(ret2))
	})
}

func TestInsertProbabilitiesAndQPS(t *testing.T) {
	withMemorySamplingStore(func(samplingStore *SamplingStore) {
		assert.NoError(t, samplingStore.InsertProbabilitiesAndQPS("dell11eg843d", model.ServiceOperationProbabilities{"new-srv": {"op": 0.1}}, model.ServiceOperationQPS{"new-srv": {"op": 4}}))
		assert.Equal(t, 1, len(samplingStore.probabilitiesAndQPS))
		// Insert one more
		assert.NoError(t, samplingStore.InsertProbabilitiesAndQPS("lncol73", model.ServiceOperationProbabilities{"my-app": {"hello": 0.3}}, model.ServiceOperationQPS{"new-srv": {"op": 7}}))
		assert.Equal(t, 2, len(samplingStore.probabilitiesAndQPS))
	})
}

func TestGetLatestProbability(t *testing.T) {
	withMemorySamplingStore(func(samplingStore *SamplingStore) {
		// No priod data
		ret, err := samplingStore.GetLatestProbabilities()
		assert.NoError(t, err)
		assert.Empty(t, ret)
	})

	withPopulatedSamplingStore(func(samplingStore *SamplingStore) {
		// With some pregenerated data
		ret, err := samplingStore.GetLatestProbabilities()
		assert.NoError(t, err)
		assert.Equal(t, ret, model.ServiceOperationProbabilities{"svc-2": {"op-3": 0.003}})
		assert.NoError(t, samplingStore.InsertProbabilitiesAndQPS("utfhyolf", model.ServiceOperationProbabilities{"another-service": {"hello": 0.009}}, model.ServiceOperationQPS{"new-srv": {"op": 5}}))
		ret, _ = samplingStore.GetLatestProbabilities()
		assert.NotEqual(t, ret, model.ServiceOperationProbabilities{"svc-2": {"op-3": 0.003}})
	})
}

func TestGetProbabilitiesAndQPS(t *testing.T) {
	withPopulatedSamplingStore(func(samplingStore *SamplingStore) {
		start := time.Now()
		ret, err := samplingStore.GetProbabilitiesAndQPS(start, start.Add(time.Second*time.Duration(1)))
		assert.NoError(t, err)
		assert.NotEmpty(t, ret)
		assert.Len(t, ret, 1)
		assert.Equal(t, &model.ProbabilityAndQPS{Probability: 0.008, QPS: 4.0}, ret["peta0242ac130003"][0]["svc-1"]["op-2"])
		ret, _ = samplingStore.GetProbabilitiesAndQPS(start, start)
		assert.Len(t, ret, 0)
		ret, _ = samplingStore.GetProbabilitiesAndQPS(start.Add(time.Second*time.Duration(-1)), start.Add(time.Second*time.Duration(10)))
		assert.Len(t, ret, 3)
	})
}
