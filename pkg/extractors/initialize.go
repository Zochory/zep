package extractors

import (
	"github.com/getzep/zep/pkg/models"
)

func Initialize(appState *models.AppState) {
	log.Info("Initializing extractors")

	attach := func(extractorType string, enabled bool, newExtractor func() models.Extractor) {
		if enabled {
			extractor := newExtractor()
			appState.MemoryStore.Attach(extractor)
			log.Infof("%s attached to memory store", extractorType)
		}
	}

	attach(
		"SummaryExtractor",
		appState.Config.Extractors.Summarizer.Enabled,
		func() models.Extractor { return NewSummaryExtractor() },
	)
	attach(
		"EmbeddingExtractor",
		appState.Config.Extractors.Embeddings.Enabled,
		func() models.Extractor { return NewEmbeddingExtractor() },
	)
	attach(
		"TokenCountExtractor",
		true, // TokenCounter always operates
		func() models.Extractor { return NewTokenCountExtractor() },
	)
	attach(
		"EntityExtractor",
		appState.Config.Extractors.Entities.Enabled,
		func() models.Extractor { return NewEntityExtractor() },
	)
}
