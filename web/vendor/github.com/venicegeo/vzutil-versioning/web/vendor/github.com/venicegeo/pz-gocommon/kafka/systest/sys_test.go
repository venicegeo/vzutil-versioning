// Copyright 2016, RadiantBlue Technologies, Inc.
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

// This test only works if you have a local Kafka installed somewhere.

package kafka

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/Shopify/sarama"
)

type KafkaTester struct {
	suite.Suite
	server string
}

func (suite *KafkaTester) SetupSuite() {
	suite.server = "localhost:9092"
}

func (suite *KafkaTester) TearDownSuite() {}

func TestRunSuite(t *testing.T) {
	s1 := new(KafkaTester)
	suite.Run(t, s1)
}

//=================================================================

type Closer interface {
	Close() error
}

func close(t *testing.T, c Closer) {
	assert := assert.New(t)

	err := c.Close()
	assert.NoError(err)
}

func makeTopicName() string {
	rand.Seed(int64(time.Now().Nanosecond()))
	topicName := fmt.Sprintf("test.%x", rand.Uint32())
	//log.Printf("topic: %s", topicName)
	return topicName
}

//=================================================================

func (suite *KafkaTester) Test01() {
	t := suite.T()
	assert := assert.New(t)

	const M1 = "message one"
	const M2 = "message two"

	var producer sarama.AsyncProducer
	var consumer sarama.Consumer
	var partitionConsumer sarama.PartitionConsumer

	var err error

	topic := makeTopicName()

	{
		config := sarama.NewConfig()
		config.Producer.Return.Successes = false
		config.Producer.Return.Errors = false

		producer, err = sarama.NewAsyncProducer([]string{suite.server}, config)
		assert.NoError(err)
		defer close(t, producer)

		producer.Input() <- &sarama.ProducerMessage{
			Topic: topic,
			Key:   nil,
			Value: sarama.StringEncoder(M1)}

		producer.Input() <- &sarama.ProducerMessage{
			Topic: topic,
			Key:   nil,
			Value: sarama.StringEncoder(M2)}
	}

	{
		consumer, err = sarama.NewConsumer([]string{suite.server}, nil)
		assert.NoError(err)
		defer close(t, consumer)

		partitionConsumer, err = consumer.ConsumePartition(topic, 0, 0)
		assert.NoError(err)
		defer close(t, partitionConsumer)
	}

	{
		mssg1 := <-partitionConsumer.Messages()
		//t.Logf("Consumed: offset:%d  value:%v", mssg1.Offset, string(mssg1.Value))
		mssg2 := <-partitionConsumer.Messages()
		//t.Logf("Consumed: offset:%d  value:%v", mssg2.Offset, string(mssg2.Value))

		assert.EqualValues(M1, string(mssg1.Value))
		assert.EqualValues(M2, string(mssg2.Value))
	}
}

//=================================================================

func doReads(t *testing.T, server string, topic string, numReads *int) {
	assert := assert.New(t)

	consumer, err := sarama.NewConsumer([]string{server}, nil)
	assert.NoError(err)
	defer close(t, consumer)

	partitionConsumer, err := consumer.ConsumePartition(topic, 0, 0)
	assert.NoError(err)
	defer close(t, partitionConsumer)

	for {
		<-partitionConsumer.Messages()
		//t.Logf("Consumed: offset:%d  value:%v", msg.Offset, string(msg.Value))
		*numReads++
	}

	//t.Logf("Reader done: %d", *numReads)
}

func doWrites(t *testing.T, server string, topic string, id int, count int) {
	assert := assert.New(t)

	config := sarama.NewConfig()
	config.Producer.Return.Successes = false
	config.Producer.Return.Errors = false

	producer, err := sarama.NewAsyncProducer([]string{server}, config)
	assert.NoError(err)
	defer close(t, producer)

	// TODO: handle "err := <-w.Errors():"

	for n := 0; n < count; n++ {
		producer.Input() <- &sarama.ProducerMessage{
			Topic: topic,
			Key:   nil,
			Value: sarama.StringEncoder(fmt.Sprintf("mssg %d from %d", n, id)),
		}
	}

	//t.Logf("Writer done: %d", count)
}

func (suite *KafkaTester) Test02() {
	t := suite.T()
	assert := assert.New(t)

	topic := makeTopicName()

	var numReads1, numReads2 int

	go doReads(t, suite.server, topic, &numReads1)
	go doReads(t, suite.server, topic, &numReads2)

	n := 3
	go doWrites(t, suite.server, topic, 1, n)
	go doWrites(t, suite.server, topic, 2, n)

	time.Sleep(1 * time.Second)

	t.Log(numReads1, "---")
	t.Log(numReads2, "---")

	assert.Equal(n*2, numReads1, "read1 count")

	assert.Equal(n*2, numReads2, "read2 count")
}
