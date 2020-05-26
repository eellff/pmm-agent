// pmm-agent
// Copyright 2019 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/percona/pmm/api/agentpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm-agent/utils/tests"
)

func TestMongoDBExplain(t *testing.T) {
	database := "test"
	collection := "test_col"
	id := "abcd1234"
	ctx := context.TODO()

	client := tests.OpenTestMongoDB(t)
	defer client.Database(database).Drop(ctx)

	err := prepareData(ctx, client, database, collection)
	require.NoError(t, err)

	params := &agentpb.StartActionRequest_MongoDBExplainParams{
		Dsn:   tests.GetTestMongoDBDSN(t),
		Query: `{"ns":"test.coll","op":"query","query":{"k":{"$lte":{"$numberInt":"1"}}}}`,
	}

	ex := NewMongoDBExplainAction(id, params)
	res, err := ex.Run(ctx)
	assert.Nil(t, err)

	want := map[string]interface{}{"indexFilterSet": false,
		"namespace": "admin.coll",
		"parsedQuery": map[string]interface{}{
			"k": map[string]interface{}{
				"$lte": map[string]interface{}{
					"$numberInt": "1",
				},
			},
		},
		"plannerVersion": map[string]interface{}{"$numberInt": "1"},
		"rejectedPlans":  []interface{}{},
		"winningPlan":    map[string]interface{}{"stage": "EOF"}}
	explainM := make(map[string]interface{})
	err = json.Unmarshal(res, &explainM)
	assert.Nil(t, err)
	queryPlanner, ok := explainM["queryPlanner"]
	assert.Equal(t, ok, true)
	assert.NotEmpty(t, queryPlanner)
	assert.Equal(t, want, queryPlanner)
}

func prepareData(ctx context.Context, client *mongo.Client, database, collection string) error {
	max := int64(100)
	count, _ := client.Database(database).Collection(collection).CountDocuments(ctx, nil)

	if count < max {
		for i := int64(0); i < max; i++ {
			doc := primitive.M{"f1": i, "f2": fmt.Sprintf("text_%5d", max-i)}
			if _, err := client.Database(database).Collection(collection).InsertOne(ctx, doc); err != nil {
				return err
			}
		}
	}

	return nil
}