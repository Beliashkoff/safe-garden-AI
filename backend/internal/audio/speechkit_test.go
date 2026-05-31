package audio

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stt "github.com/yandex-cloud/go-genproto/yandex/cloud/ai/stt/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// fakeRecognizer is an in-process Recognizer server used to exercise the client
// without a network or real SpeechKit endpoint.
type fakeRecognizer struct {
	stt.UnimplementedRecognizerServer
	gotAuth    string
	gotOptions *stt.StreamingOptions
	gotAudio   []byte
	final      string
	refined    string
	failWith   error
}

func (f *fakeRecognizer) RecognizeStreaming(stream grpc.BidiStreamingServer[stt.StreamingRequest, stt.StreamingResponse]) error {
	if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
		if v := md.Get("authorization"); len(v) > 0 {
			f.gotAuth = v[0]
		}
	}
	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		switch e := req.GetEvent().(type) {
		case *stt.StreamingRequest_SessionOptions:
			f.gotOptions = e.SessionOptions
		case *stt.StreamingRequest_Chunk:
			f.gotAudio = append(f.gotAudio, e.Chunk.GetData()...)
		}
	}
	if f.failWith != nil {
		return f.failWith
	}
	if f.final != "" {
		if err := stream.Send(&stt.StreamingResponse{Event: &stt.StreamingResponse_Final{
			Final: &stt.AlternativeUpdate{Alternatives: []*stt.Alternative{{Text: f.final, EndTimeMs: 4200}}},
		}}); err != nil {
			return err
		}
	}
	if f.refined != "" {
		if err := stream.Send(&stt.StreamingResponse{Event: &stt.StreamingResponse_FinalRefinement{
			FinalRefinement: &stt.FinalRefinement{Type: &stt.FinalRefinement_NormalizedText{
				NormalizedText: &stt.AlternativeUpdate{Alternatives: []*stt.Alternative{{Text: f.refined, EndTimeMs: 4300}}},
			}},
		}}); err != nil {
			return err
		}
	}
	return nil
}

func dialFake(t *testing.T, srv *fakeRecognizer) *SpeechKitTranscriber {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	gs := grpc.NewServer()
	stt.RegisterRecognizerServer(gs, srv)
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(gs.Stop)

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return newSpeechKitWithClient(stt.NewRecognizerClient(conn), "test-key", "general")
}

func TestSpeechKit_Transcribe_PrefersRefinedAndSendsOptions(t *testing.T) {
	t.Parallel()
	srv := &fakeRecognizer{final: "вянут помидоры", refined: "Вянут помидоры."}
	tr := dialFake(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	audio := make([]byte, 40*1024) // > audioChunkBytes -> several chunks
	for i := range audio {
		audio[i] = byte(i)
	}

	res, err := tr.Transcribe(ctx, audio, "ru-RU")
	require.NoError(t, err)
	assert.Equal(t, "Вянут помидоры.", res.Text)
	assert.Equal(t, int64(4300), res.DurationMs)

	assert.Equal(t, "Api-Key test-key", srv.gotAuth)
	assert.Equal(t, audio, srv.gotAudio)

	require.NotNil(t, srv.gotOptions)
	rec := srv.gotOptions.GetRecognitionModel()
	require.NotNil(t, rec)
	assert.Equal(t, stt.ContainerAudio_OGG_OPUS, rec.GetAudioFormat().GetContainerAudio().GetContainerAudioType())
	assert.Equal(t, []string{"ru-RU"}, rec.GetLanguageRestriction().GetLanguageCode())
	assert.Equal(t, stt.RecognitionModelOptions_FULL_DATA, rec.GetAudioProcessingType())
}

func TestSpeechKit_Transcribe_FallsBackToFinal(t *testing.T) {
	t.Parallel()
	tr := dialFake(t, &fakeRecognizer{final: "только финал"})
	res, err := tr.Transcribe(context.Background(), []byte("audio-bytes"), "ru-RU")
	require.NoError(t, err)
	assert.Equal(t, "только финал", res.Text)
	assert.Equal(t, int64(4200), res.DurationMs)
}

func TestSpeechKit_Transcribe_EmptyResult(t *testing.T) {
	t.Parallel()
	tr := dialFake(t, &fakeRecognizer{}) // server emits no events
	_, err := tr.Transcribe(context.Background(), []byte("audio-bytes"), "ru-RU")
	require.ErrorIs(t, err, ErrEmptyResult)
}

func TestSpeechKit_Transcribe_ServerError(t *testing.T) {
	t.Parallel()
	tr := dialFake(t, &fakeRecognizer{failWith: status.Error(codes.Internal, "boom")})
	_, err := tr.Transcribe(context.Background(), []byte("audio-bytes"), "ru-RU")
	require.Error(t, err)
}

func TestSpeechKit_Transcribe_RejectsEmptyAudio(t *testing.T) {
	t.Parallel()
	tr := dialFake(t, &fakeRecognizer{final: "x"})
	_, err := tr.Transcribe(context.Background(), nil, "ru-RU")
	require.Error(t, err)
}

func TestNewSpeechKit_RequiresAPIKey(t *testing.T) {
	t.Parallel()
	_, err := NewSpeechKitTranscriber(&Config{SpeechKitEndpoint: "stt.api.cloud.yandex.net:443"})
	require.Error(t, err)
}
