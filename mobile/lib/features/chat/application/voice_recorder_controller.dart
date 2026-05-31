import 'dart:async';
import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path_provider/path_provider.dart';

import '../../../core/network/api_exception.dart';
import '../data/audio_ports.dart';
import '../data/media_cache.dart';
import '../data/media_ports.dart';
import '../data/upload_api.dart';
import 'chat_controller.dart';

/// Phase of the voice composer: nothing (idle), capturing (recording), recorded
/// and awaiting confirmation (preview), or sending (uploading).
enum VoicePhase { idle, recording, preview, uploading }

/// The result of attempting to start a recording — the UI maps it to a prompt.
enum VoiceStartResult {
  started,
  permissionDenied,
  permissionPermanentlyDenied,
  failed,
}

/// Immutable voice composer state.
class VoiceRecorderState {
  const VoiceRecorderState({
    this.phase = VoicePhase.idle,
    this.elapsedMs = 0,
    this.amplitude = 0,
    this.filePath,
    this.cancelArmed = false,
  });

  final VoicePhase phase;
  final int elapsedMs;
  final double amplitude; // 0..1 input level
  final String? filePath;
  final bool cancelArmed; // dragged left far enough to cancel on release

  bool get isActive =>
      phase == VoicePhase.recording || phase == VoicePhase.preview;

  VoiceRecorderState copyWith({
    VoicePhase? phase,
    int? elapsedMs,
    double? amplitude,
    String? filePath,
    bool? cancelArmed,
  }) {
    return VoiceRecorderState(
      phase: phase ?? this.phase,
      elapsedMs: elapsedMs ?? this.elapsedMs,
      amplitude: amplitude ?? this.amplitude,
      filePath: filePath ?? this.filePath,
      cancelArmed: cancelArmed ?? this.cancelArmed,
    );
  }
}

/// Drives the hold-to-record voice composer: permission → record (timer +
/// level) → preview → presign → PUT → cache, then hands the audio key to
/// [ChatController.sendMessage]. Mirrors the photo upload pipeline; recording
/// state lives here.
class VoiceRecorderController extends Notifier<VoiceRecorderState> {
  /// Maximum recording length (SPEC F5). Auto-stops to preview at this point.
  static const int maxDurationMs = 60000;

  /// Leftward drag (px from the press origin) that arms cancel-on-release.
  static const double cancelThresholdPx = 80;

  Timer? _ticker;
  StreamSubscription<double>? _ampSub;
  bool _stopRequested = false; // release arrived before start() finished

  AudioRecorderPort get _recorder => ref.read(audioRecorderPortProvider);
  PermissionPort get _perms => ref.read(permissionPortProvider);
  UploadApi get _uploads => ref.read(uploadApiProvider);
  MediaCache get _media => ref.read(mediaCacheProvider);

  @override
  VoiceRecorderState build() {
    ref.onDispose(_stopTickers);
    return const VoiceRecorderState();
  }

  Future<VoiceStartResult> startRecording() async {
    if (state.phase != VoicePhase.idle) {
      return VoiceStartResult.failed;
    }
    final outcome = await _perms.ensureMicrophone();
    switch (outcome) {
      case PermissionOutcome.denied:
        return VoiceStartResult.permissionDenied;
      case PermissionOutcome.permanentlyDenied:
        return VoiceStartResult.permissionPermanentlyDenied;
      case PermissionOutcome.granted:
        break;
    }
    _stopRequested = false;
    try {
      final dir = await getTemporaryDirectory();
      final path =
          '${dir.path}/voice_${DateTime.now().microsecondsSinceEpoch}.m4a';
      await _recorder.start(path);
      if (_stopRequested) {
        // Released during the async start — discard the just-started recording.
        await _recorder.cancel();
        state = const VoiceRecorderState();
        return VoiceStartResult.failed;
      }
      state = VoiceRecorderState(phase: VoicePhase.recording, filePath: path);
      _ampSub = _recorder.amplitude().listen(
        (level) => state = state.copyWith(amplitude: level),
      );
      _ticker = Timer.periodic(const Duration(milliseconds: 100), (_) {
        final next = state.elapsedMs + 100;
        state = state.copyWith(elapsedMs: next);
        if (next >= maxDurationMs) {
          unawaited(stopToPreview());
        }
      });
      return VoiceStartResult.started;
    } on Object {
      state = const VoiceRecorderState();
      return VoiceStartResult.failed;
    }
  }

  /// Updates the cancel-armed flag from the current leftward drag distance.
  void updateDrag(double dxFromOrigin) {
    if (state.phase != VoicePhase.recording) {
      return;
    }
    final armed = dxFromOrigin <= -cancelThresholdPx;
    if (armed != state.cancelArmed) {
      state = state.copyWith(cancelArmed: armed);
    }
  }

  /// Stops recording and moves to preview (kept for play-before-send).
  Future<void> stopToPreview() async {
    if (state.phase != VoicePhase.recording) {
      _stopRequested = true; // start() may still be in flight
      return;
    }
    _stopTickers();
    final path = await _recorder.stop();
    if (path == null) {
      state = const VoiceRecorderState();
      return;
    }
    state = state.copyWith(
      phase: VoicePhase.preview,
      filePath: path,
      cancelArmed: false,
    );
  }

  /// Cancels an in-progress recording or discards a preview.
  Future<void> cancelRecording() async {
    switch (state.phase) {
      case VoicePhase.recording:
        _stopTickers();
        await _recorder.cancel();
      case VoicePhase.preview:
        await _deletePreviewFile();
      case VoicePhase.idle:
      case VoicePhase.uploading:
        return;
    }
    state = const VoiceRecorderState();
  }

  Future<void> discardPreview() async {
    if (state.phase != VoicePhase.preview) {
      return;
    }
    await _deletePreviewFile();
    state = const VoiceRecorderState();
  }

  /// Uploads the previewed recording and sends it as a voice message. Returns
  /// false (keeping the preview, so the user can retry) on upload failure.
  Future<bool> sendPreview() async {
    if (state.phase != VoicePhase.preview) {
      return false;
    }
    final path = state.filePath;
    if (path == null) {
      return false;
    }
    final durationMs = state.elapsedMs;
    state = state.copyWith(phase: VoicePhase.uploading);
    try {
      final file = File(path);
      final bytes = await file.readAsBytes();
      final presign = await _uploads.presignPut(
        contentType: 'audio/m4a',
        sizeBytes: bytes.length,
      );
      await _uploads.putObject(
        url: presign.url,
        headers: presign.headers,
        bytes: bytes,
      );
      await _media.store(presign.key, file);
      await ref
          .read(chatControllerProvider.notifier)
          .sendMessage(
            '',
            audioStorageKey: presign.key,
            audioDurationMs: durationMs,
          );
      state = const VoiceRecorderState();
      return true;
    } on AppException {
      state = state.copyWith(phase: VoicePhase.preview);
      return false;
    }
  }

  void _stopTickers() {
    _ticker?.cancel();
    _ticker = null;
    unawaited(_ampSub?.cancel());
    _ampSub = null;
  }

  Future<void> _deletePreviewFile() async {
    final path = state.filePath;
    if (path == null) {
      return;
    }
    final f = File(path);
    if (await f.exists()) {
      await f.delete();
    }
  }
}

final voiceRecorderProvider =
    NotifierProvider<VoiceRecorderController, VoiceRecorderState>(
      VoiceRecorderController.new,
    );
