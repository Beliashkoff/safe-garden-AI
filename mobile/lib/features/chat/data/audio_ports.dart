import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:just_audio/just_audio.dart';
import 'package:record/record.dart';

/// Records microphone audio to a file. A narrow port so the voice controller can
/// be unit-tested without the platform plugin.
abstract interface class AudioRecorderPort {
  /// Starts recording to [path] (AAC-LC in an m4a container, mono, 16 kHz).
  Future<void> start(String path);

  /// Stops recording and returns the file path, or null on failure.
  Future<String?> stop();

  /// Stops and discards the recording (removes the file).
  Future<void> cancel();

  /// Normalised input level (0..1), sampled every [interval]. For the meter.
  Stream<double> amplitude({
    Duration interval = const Duration(milliseconds: 200),
  });

  Future<void> dispose();
}

/// `record`-backed recorder. Outputs AAC-LC/m4a, which the backend accepts
/// (audio/m4a) and converts to OggOpus for SpeechKit.
class RecordAudioRecorder implements AudioRecorderPort {
  RecordAudioRecorder([AudioRecorder? recorder])
    : _recorder = recorder ?? AudioRecorder();

  final AudioRecorder _recorder;

  /// dBFS floor mapped to level 0 (silence ~ -45 dBFS).
  static const double _minDb = -45;

  @override
  Future<void> start(String path) => _recorder.start(
    const RecordConfig(
      encoder: AudioEncoder.aacLc,
      numChannels: 1,
      sampleRate: 16000,
    ),
    path: path,
  );

  @override
  Future<String?> stop() => _recorder.stop();

  @override
  Future<void> cancel() => _recorder.cancel();

  @override
  Stream<double> amplitude({
    Duration interval = const Duration(milliseconds: 200),
  }) {
    return _recorder.onAmplitudeChanged(interval).map((a) {
      final level = (a.current - _minDb) / (0 - _minDb);
      return level.clamp(0.0, 1.0);
    });
  }

  @override
  Future<void> dispose() => _recorder.dispose();
}

/// Plays a local audio file. Narrow port over `just_audio`.
abstract interface class AudioPlayerPort {
  /// Loads [path]; returns its duration if known.
  Future<Duration?> setFilePath(String path);
  Future<void> play();
  Future<void> pause();
  Future<void> stop();
  Future<void> seek(Duration position);
  Stream<Duration> position();
  Stream<bool> playing();
  Duration? get duration;
  Future<void> dispose();
}

class JustAudioPlayer implements AudioPlayerPort {
  JustAudioPlayer([AudioPlayer? player]) : _player = player ?? AudioPlayer();

  final AudioPlayer _player;

  @override
  Future<Duration?> setFilePath(String path) => _player.setFilePath(path);

  @override
  Future<void> play() => _player.play();

  @override
  Future<void> pause() => _player.pause();

  @override
  Future<void> stop() => _player.stop();

  @override
  Future<void> seek(Duration position) => _player.seek(position);

  @override
  Stream<Duration> position() => _player.positionStream;

  @override
  Stream<bool> playing() => _player.playingStream;

  @override
  Duration? get duration => _player.duration;

  @override
  Future<void> dispose() => _player.dispose();
}

final audioRecorderPortProvider = Provider<AudioRecorderPort>((ref) {
  final rec = RecordAudioRecorder();
  ref.onDispose(rec.dispose);
  return rec;
});

/// Builds a fresh [AudioPlayerPort]. Each voice bubble / preview owns and
/// disposes its own player, so they don't share playback state. Overridable in
/// tests to return a fake.
typedef AudioPlayerFactory = AudioPlayerPort Function();

final audioPlayerFactoryProvider = Provider<AudioPlayerFactory>(
  (ref) => JustAudioPlayer.new,
);
