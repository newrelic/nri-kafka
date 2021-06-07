package integration

/*
func TestKafkaIntegration_zookeeper_with_topicSourceZookeeper(t *testing.T) {
	zookeeperDiscoverConfigTopicSourceZookeeper := func(command []string) []string {
		return append(zookeeperDiscoverConfig(command), "--topic_source", "zookeeper")
	}

	stdout, stderr, err := runIntegration(t, zookeeperDiscoverConfigTopicSourceZookeeper)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
	for _, topic := range topicNames {
		assert.Contains(t, stdout, topic, fmt.Sprintf("The output doesn't have the topic %s", topic))
	}
}

func TestKafkaIntegration_bootstrap(t *testing.T) {
	stdout, stderr, err := runIntegration(t, bootstrapDiscoverConfig)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
	for _, topic := range topicNames {
		assert.Contains(t, stdout, topic, fmt.Sprintf("The output doesn't have the topic %s", topic))
	}
}

func TestKafkaIntegration_bootstrap_with_topicSourceZookeeper(t *testing.T) {
	bootstrapDiscoverConfigConfigTopicSourceZookeeper := func(command []string) []string {
		return append(bootstrapDiscoverConfig(command), "--topic_source", "zookeeper")
	}

	stdout, stderr, err := runIntegration(t, bootstrapDiscoverConfigConfigTopicSourceZookeeper)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
	for _, topic := range topicNames {
		assert.Contains(t, stdout, topic, fmt.Sprintf("The output doesn't have the topic %s", topic))
	}
}

func TestKafkaIntegration_bootstrap_topicBucket(t *testing.T) {
	bootstrapDiscoverConfigConfigTopicBucket := func(command []string) []string {
		return append(bootstrapDiscoverConfig(command), "--topic_bucket", "3/3")
	}

	stdout, stderr, err := runIntegration(t, bootstrapDiscoverConfigConfigTopicBucket)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")

	var topicsCount int
	for _, topic := range topicNames {
		if strings.Contains(stdout, topic) {
			topicsCount++
		}
	}
	assert.Equal(t, 1, topicsCount)
}

func TestKafkaIntegration_bootstrap_localOnlyCollection(t *testing.T) {
	bootstrapDiscoverConfigLocalOnlyCollection := func(command []string) []string {
		return append(bootstrapDiscoverConfig(command), "--local_only_collection")
	}

	stdout, stderr, err := runIntegration(t, bootstrapDiscoverConfigLocalOnlyCollection)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema-only-local.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
	for _, topic := range topicNames {
		assert.Contains(t, stdout, topic, fmt.Sprintf("The output doesn't have the topic %s", topic))
	}
}

func TestKafkaIntegration_bootstrap_metrics(t *testing.T) {
	bootstrapDiscoverConfigMetrics := func(command []string) []string {
		return append(bootstrapDiscoverConfig(command), "--metrics")
	}

	stdout, stderr, err := runIntegration(t, bootstrapDiscoverConfigMetrics)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema-metrics.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
	for _, topic := range topicNames {
		assert.Contains(t, stdout, topic, fmt.Sprintf("The output doesn't have the topic %s", topic))
	}
}

func TestKafkaIntegration_bootstrap_inventory(t *testing.T) {
	bootstrapDiscoverConfigInventory := func(command []string) []string {
		return append(bootstrapDiscoverConfig(command), "--inventory")
	}

	stdout, stderr, err := runIntegration(t, bootstrapDiscoverConfigInventory)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema-inventory.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
	for _, topic := range topicNames {
		assert.Contains(t, stdout, topic, fmt.Sprintf("The output doesn't have the topic %s", topic))
	}
}
*/
