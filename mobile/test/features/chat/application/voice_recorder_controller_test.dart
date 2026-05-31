import 'dart:async';
import 'dart:io';

import 'package:agronom_ai/features/chat/application/chat_controller.dart';
import 'package:agronom_ai/features/chat/application/voice_recorder_controller.dart';
import 'package:agronom_ai/features/chat/data/audio_ports.dart';
import 'package:agronom_ai/features/chat/data/media_cache.dart';
import 'package:agronom_ai/features/chat/data/media_ports.dart';
import 'package:agronom_ai/features/chat/data/upload_api.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:plugin_platform_interface/plugin_platform_interface.dart';

import '../chat_test_helpers.dart';

class _FakePathProvider extends PathProviderPlatform
    with MockPlatformInterfaceMixin {
  @override
  Future<String?> getTemporaryPath() async => Directory.systemTemp.path;
}

class _FakeRecorder implements AudioRecorderPort {
  String? startedPath;
  bool canceled = false;
  final _amp = StreamController<double>.broadcast();

  @override
  Future<void> start(String path) async {
    startedPath = path;
    await File(path).writeAsBytes(const [1, 2, 3, 4]);
  }

  @override
  Future<String?> stop() async => startedPath;

  @override
  Future<void> cancel() async {
    canceled = true;
    final p = startedPath;
    if (p != null) {
      final f = File(p);
      if (await f.exists()) {
        await f.delete();
      }
    }
  }

  @override
  Stream<double> amplitude({
    Duration interval = const Duration(milliseconds: 200),
  }) => _amp.stream;

  @override
  Future<void> dispose() => _amp.close();
}

class _FakeMicPermission implements PermissionPort {
  _FakeMicPermission(this.outcome);
  PermissionOutcome outcome;

  @override
  Future<PermissionOutcome> ensureCamera() async => outcome;
  @override
  Future<PermissionOutcome> ensurePhotos() async => outcome;
  @override
  Future<PermissionOutcome> ensureMicrophone() async => outcome;
  @override
  Future<void> openSettings() async {}
}

class _FakeUploadApi implements UploadApi {
  int putCalls = 0;
  String? lastContentType;

  @override
  Future<PresignResult> presignPut({
    required String contentType,
    required int sizeBytes,
  }) async {
    lastContentType = contentType;
    return const PresignResult(
      url: 'http://put',
      key: 'u/a/audio/1.m4a',
      headers: {},
    );
  }

  @override
  Future<void> putObject({
    required String url,
    required Map<String, String> headers,
    required List<int> bytes,
    ProgressCallback? onSendProgress,
    CancelToken? cancelToken,
  }) async {
    putCalls++;
  }

  @override
  Future<ViewResult> presignView({required String storageKey}) async =>
      const ViewResult('http://get');

  @override
  Future<List<int>> downloadBytes({
    required String url,
    CancelToken? cancelToken,
  }) async => const [];
}

class _FakeMediaCache implements MediaCache {
  final List<String> stored = [];

  @override
  Future<File?> fileForKey(String storageKey) async => null;
  @override
  Future<File> store(String storageKey, File source) async {
    stored.add(storageKey);
    return source;
  }

  @override
  Future<File> ensure(String storageKey) async => throw UnimplementedError();
  @override
  Future<void> clear() async {}
}

void main() {
  setUp(() => PathProviderPlatform.instance = _FakePathProvider());

  ProviderContainer makeContainer(
    FakeChatRepository repo, {
    required PermissionOutcome perm,
    required AudioRecorderPort recorder,
    UploadApi? upload,
    MediaCache? media,
  }) {
    final container = ProviderContainer(
      overrides: [
        ...chatTestOverrides(repo),
        permissionPortProvider.overrideWithValue(_FakeMicPermission(perm)),
        audioRecorderPortProvider.overrideWithValue(recorder),
        uploadApiProvider.overrideWithValue(upload ?? _FakeUploadApi()),
        mediaCacheProvider.overrideWithValue(media ?? _FakeMediaCache()),
      ],
    );
    addTearDown(container.dispose);
    return container;
  }

  test('permission denied keeps the recorder idle', () async {
    final container = makeContainer(
      FakeChatRepository(),
      perm: PermissionOutcome.denied,
      recorder: _FakeRecorder(),
    );
    final notifier = container.read(voiceRecorderProvider.notifier);

    final result = await notifier.startRecording();

    expect(result, VoiceStartResult.permissionDenied);
    expect(container.read(voiceRecorderProvider).phase, VoicePhase.idle);
  });

  test('records then moves to preview', () async {
    final recorder = _FakeRecorder();
    final container = makeContainer(
      FakeChatRepository(),
      perm: PermissionOutcome.granted,
      recorder: recorder,
    );
    final notifier = container.read(voiceRecorderProvider.notifier);

    expect(await notifier.startRecording(), VoiceStartResult.started);
    expect(container.read(voiceRecorderProvider).phase, VoicePhase.recording);

    await notifier.stopToPreview();
    final state = container.read(voiceRecorderProvider);
    expect(state.phase, VoicePhase.preview);
    expect(state.filePath, isNotNull);
    expect(recorder.canceled, isFalse);

    final f = File(state.filePath!);
    if (await f.exists()) {
      await f.delete();
    }
  });

  test('cancel discards the recording', () async {
    final recorder = _FakeRecorder();
    final container = makeContainer(
      FakeChatRepository(),
      perm: PermissionOutcome.granted,
      recorder: recorder,
    );
    final notifier = container.read(voiceRecorderProvider.notifier);

    await notifier.startRecording();
    await notifier.cancelRecording();

    expect(container.read(voiceRecorderProvider).phase, VoicePhase.idle);
    expect(recorder.canceled, isTrue);
  });

  test('sendPreview uploads and dispatches a voice message', () async {
    final repo = FakeChatRepository();
    final recorder = _FakeRecorder();
    final upload = _FakeUploadApi();
    final media = _FakeMediaCache();
    final container = makeContainer(
      repo,
      perm: PermissionOutcome.granted,
      recorder: recorder,
      upload: upload,
      media: media,
    );
    await container.read(chatControllerProvider.future); // build chat state
    final notifier = container.read(voiceRecorderProvider.notifier);

    await notifier.startRecording();
    await notifier.stopToPreview();
    final ok = await notifier.sendPreview();
    await pumpEventQueue();

    expect(ok, isTrue);
    expect(upload.lastContentType, 'audio/m4a');
    expect(upload.putCalls, 1);
    expect(media.stored, ['u/a/audio/1.m4a']);
    expect(repo.lastSendAudioKey, 'u/a/audio/1.m4a');
    expect(container.read(voiceRecorderProvider).phase, VoicePhase.idle);
  });
}
