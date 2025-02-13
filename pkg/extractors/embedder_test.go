package extractors

import (
	"context"
	"math"
	"testing"

	"github.com/getzep/zep/pkg/llms"
	"github.com/getzep/zep/pkg/memorystore"
	"github.com/getzep/zep/pkg/models"
	"github.com/getzep/zep/pkg/testutils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestEmbeddingExtractor_Extract(t *testing.T) {
	ctx := context.Background()

	// Force embedding to be enabled
	viper.Set("extractor.embeddings.enabled", true)

	db := memorystore.NewPostgresConn(testutils.GetDSN())
	memorystore.CleanDB(t, db)

	cfg := testutils.NewTestConfig()

	appState := &models.AppState{Config: cfg}
	store, err := memorystore.NewPostgresMemoryStore(appState, db)
	assert.NoError(t, err)
	appState.MemoryStore = store
	appState.OpenAIClient = llms.NewOpenAIRetryClient(cfg)

	sessionID, err := testutils.GenerateRandomSessionID(16)
	assert.NoError(t, err)

	testMessages := testutils.TestMessages[:5]

	// Add new messages using appState.MemoryStore.PutMemory
	err = store.PutMemory(ctx, appState, sessionID, &models.Memory{Messages: testMessages}, true)
	assert.NoError(t, err)

	// Get messages that are missing embeddings using appState.MemoryStore.GetMessageVectors
	memories, err := store.GetMemory(ctx, appState, sessionID, 0)
	assert.NoError(t, err)
	assert.True(t, len(memories.Messages) == len(testMessages))

	unembeddedMessages := memories.Messages
	// Create messageEvent. We only need to pass the sessionID
	messageEvent := &models.MessageEvent{
		SessionID: sessionID,
		Messages:  unembeddedMessages,
	}

	texts := make([]string, len(unembeddedMessages))
	for i, r := range unembeddedMessages {
		texts[i] = r.Content
	}

	embeddings, err := llms.EmbedMessages(ctx, appState, texts)
	assert.NoError(t, err)

	expectedEmbeddingRecords := make([]models.Embeddings, len(unembeddedMessages))
	for i, r := range unembeddedMessages {
		expectedEmbeddingRecords[i] = models.Embeddings{
			TextUUID:  r.UUID,
			Text:      r.Content,
			Embedding: embeddings[i].Embedding,
		}
	}

	embeddingExtractor := NewEmbeddingExtractor()
	err = embeddingExtractor.Extract(ctx, appState, messageEvent)
	assert.NoError(t, err)

	embeddedMessages, err := store.GetMessageVectors(
		ctx,
		appState,
		messageEvent.SessionID,
	)
	assert.NoError(t, err)

	assert.Equal(t, len(expectedEmbeddingRecords), len(embeddedMessages))

	// Test if the length of embeddedMessages is equal to the length of messageEvent.Messages
	for i, r := range embeddedMessages {
		assert.Equal(t, expectedEmbeddingRecords[i].TextUUID, r.TextUUID)
		assert.Equal(t, expectedEmbeddingRecords[i].Text, r.Text)
		compareFloat32Vectors(t, expectedEmbeddingRecords[i].Embedding, r.Embedding, 0.001)
	}
}

// compareFloat32Vectors compares two float32 vectors, asserting that their values are within the given variance.
func compareFloat32Vectors(t *testing.T, a, b []float32, variance float32) {
	t.Helper()

	if len(a) != len(b) {
		t.Fatalf("Vectors have different lengths: len(a)=%d, len(b)=%d", len(a), len(b))
	}

	for i := 0; i < len(a); i++ {
		diff := float32(math.Abs(float64(a[i] - b[i])))
		if diff > variance {
			t.Fatalf(
				"Vectors differ at index %d: a=%v, b=%v, diff=%v, variance=%v",
				i,
				a[i],
				b[i],
				diff,
				variance,
			)
		}
	}
}
