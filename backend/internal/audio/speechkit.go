package audio

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"strings"

	stt "github.com/yandex-cloud/go-genproto/yandex/cloud/ai/stt/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// audioChunkBytes is the size of each audio frame sent over the stream. The
// pre-recorded OggOpus is split into chunks purely to fit gRPC message limits;
// the value is not latency-sensitive here.
const audioChunkBytes = 16 * 1024

// SpeechKitTranscriber recognizes speech via Yandex SpeechKit v3 streaming
// (Recognizer.RecognizeStreaming). Yandex Cloud is reachable from the RU
// backend directly, so unlike Anthropic this does not go through the worker.
type SpeechKitTranscriber struct {
	conn   *grpc.ClientConn // nil when a client is injected (tests)
	client stt.RecognizerClient
	apiKey string
	folder string
	model  string
}

// NewSpeechKitTranscriber dials SpeechKit over TLS. The connection is lazy
// (grpc.NewClient connects on first RPC) and reused across calls; call Close on
// shutdown.
func NewSpeechKitTranscriber(cfg *Config) (*SpeechKitTranscriber, error) {
	if cfg.SpeechKitAPIKey == "" {
		return nil, fmt.Errorf("audio.speechkit: SPEECHKIT_API_KEY is empty")
	}
	if cfg.SpeechKitEndpoint == "" {
		return nil, fmt.Errorf("audio.speechkit: SPEECHKIT_ENDPOINT is empty")
	}
	creds := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
	conn, err := grpc.NewClient(cfg.SpeechKitEndpoint, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("audio.speechkit: dial %s: %w", cfg.SpeechKitEndpoint, err)
	}
	t := newSpeechKitWithClient(stt.NewRecognizerClient(conn), cfg.SpeechKitAPIKey, cfg.SpeechKitModel)
	t.conn = conn
	t.folder = cfg.SpeechKitFolderID
	return t, nil
}

func newSpeechKitWithClient(client stt.RecognizerClient, apiKey, model string) *SpeechKitTranscriber {
	if model == "" {
		model = "general"
	}
	return &SpeechKitTranscriber{client: client, apiKey: apiKey, model: model}
}

var _ Transcriber = (*SpeechKitTranscriber)(nil)

// Close releases the gRPC connection. Safe to call on a zero/injected client.
func (t *SpeechKitTranscriber) Close() error {
	if t.conn != nil {
		return t.conn.Close()
	}
	return nil
}

func (t *SpeechKitTranscriber) Transcribe(ctx context.Context, oggOpus []byte, lang string) (Result, error) {
	if len(oggOpus) == 0 {
		return Result{}, fmt.Errorf("audio.speechkit: empty audio")
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Api-Key "+t.apiKey)
	if t.folder != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-folder-id", t.folder)
	}

	stream, err := t.client.RecognizeStreaming(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("audio.speechkit: open stream: %w", err)
	}

	if err := stream.Send(sessionOptions(t.model, lang)); err != nil {
		return Result{}, fmt.Errorf("audio.speechkit: send options: %w", err)
	}

	// Send audio while reading results concurrently: the server may emit events
	// before all chunks are sent. The buffered channel lets the sender finish
	// even if the receive loop exits early on error.
	sendErr := make(chan error, 1)
	go func() { sendErr <- sendChunks(stream, oggOpus) }()

	var finals, refined []string
	var durationMs int64
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Result{}, fmt.Errorf("audio.speechkit: recv: %w", err)
		}
		switch e := resp.GetEvent().(type) {
		case *stt.StreamingResponse_Final:
			if txt, end := firstAlternative(e.Final); txt != "" {
				finals = append(finals, txt)
				durationMs = max64(durationMs, end)
			}
		case *stt.StreamingResponse_FinalRefinement:
			if txt, end := firstAlternative(e.FinalRefinement.GetNormalizedText()); txt != "" {
				refined = append(refined, txt)
				durationMs = max64(durationMs, end)
			}
		}
	}
	if err := <-sendErr; err != nil {
		return Result{}, fmt.Errorf("audio.speechkit: send audio: %w", err)
	}

	// Prefer normalized (refined) text when present; otherwise fall back to the
	// raw finals.
	parts := refined
	if len(parts) == 0 {
		parts = finals
	}
	text := strings.TrimSpace(strings.Join(parts, " "))
	if text == "" {
		return Result{}, ErrEmptyResult
	}
	return Result{Text: text, DurationMs: durationMs}, nil
}

// sessionOptions builds the first streaming message: tell SpeechKit the audio is
// OggOpus, restrict the language, and run in FULL_DATA mode (recognize once all
// audio is received — we are transcribing a finished recording, not live mic).
func sessionOptions(model, lang string) *stt.StreamingRequest {
	rec := &stt.RecognitionModelOptions{
		Model: model,
		AudioFormat: &stt.AudioFormatOptions{
			AudioFormat: &stt.AudioFormatOptions_ContainerAudio{
				ContainerAudio: &stt.ContainerAudio{
					ContainerAudioType: stt.ContainerAudio_OGG_OPUS,
				},
			},
		},
		TextNormalization: &stt.TextNormalizationOptions{
			TextNormalization: stt.TextNormalizationOptions_TEXT_NORMALIZATION_ENABLED,
		},
		AudioProcessingType: stt.RecognitionModelOptions_FULL_DATA,
	}
	if lang != "" {
		rec.LanguageRestriction = &stt.LanguageRestrictionOptions{
			RestrictionType: stt.LanguageRestrictionOptions_WHITELIST,
			LanguageCode:    []string{lang},
		}
	}
	return &stt.StreamingRequest{
		Event: &stt.StreamingRequest_SessionOptions{
			SessionOptions: &stt.StreamingOptions{RecognitionModel: rec},
		},
	}
}

func sendChunks(stream grpc.BidiStreamingClient[stt.StreamingRequest, stt.StreamingResponse], audio []byte) error {
	for off := 0; off < len(audio); off += audioChunkBytes {
		end := off + audioChunkBytes
		if end > len(audio) {
			end = len(audio)
		}
		req := &stt.StreamingRequest{
			Event: &stt.StreamingRequest_Chunk{
				Chunk: &stt.AudioChunk{Data: audio[off:end]},
			},
		}
		if err := stream.Send(req); err != nil {
			return err
		}
	}
	return stream.CloseSend()
}

func firstAlternative(u *stt.AlternativeUpdate) (string, int64) {
	if u == nil {
		return "", 0
	}
	alts := u.GetAlternatives()
	if len(alts) == 0 {
		return "", 0
	}
	return alts[0].GetText(), alts[0].GetEndTimeMs()
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
