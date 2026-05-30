import 'dart:io';

import 'package:agronom_ai/core/network/api_exception.dart';
import 'package:agronom_ai/features/chat/application/chat_controller.dart';
import 'package:agronom_ai/features/chat/application/message_composer.dart';
import 'package:agronom_ai/features/chat/data/media_cache.dart';
import 'package:agronom_ai/features/chat/data/media_ports.dart';
import 'package:agronom_ai/features/chat/data/upload_api.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../chat_test_helpers.dart';

class _FakePicker implements ImagePickerPort {
  _FakePicker({this.cameraPath, this.galleryPaths = const []});

  String? cameraPath;
  List<String> galleryPaths;
  int? lastGalleryLimit;

  @override
  Future<String?> pickCamera() async => cameraPath;

  @override
  Future<List<String>> pickGallery({required int limit}) async {
    lastGalleryLimit = limit;
    return galleryPaths.take(limit).toList();
  }
}

/// Echoes the source path back as the "compressed" file (no real compression).
class _FakeCompressor implements ImageCompressorPort {
  @override
  Future<File> compress(String sourcePath) async => File(sourcePath);
}

class _FakePermissions implements PermissionPort {
  _FakePermissions(this.outcome);

  PermissionOutcome outcome;
  int openSettingsCalls = 0;

  @override
  Future<PermissionOutcome> ensureCamera() async => outcome;

  @override
  Future<PermissionOutcome> ensurePhotos() async => outcome;

  @override
  Future<void> openSettings() async => openSettingsCalls++;
}

class _FakeUploadApi implements UploadApi {
  _FakeUploadApi({this.fail = false});

  bool fail;
  int putCalls = 0;
  final List<int> presignSizes = [];

  @override
  Future<PresignResult> presignPut({
    required String contentType,
    required int sizeBytes,
  }) async {
    presignSizes.add(sizeBytes);
    return PresignResult(
      url: 'http://put',
      key: 'u/a/img/${presignSizes.length}.jpg',
      headers: const {'Content-Type': 'image/jpeg'},
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
    onSendProgress?.call(bytes.length, bytes.length);
    if (fail) {
      throw const NetworkException();
    }
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
  Future<File> ensure(String storageKey) async =>
      throw UnimplementedError();

  @override
  Future<void> clear() async {}
}

ProviderContainer _container({
  required FakeChatRepository repo,
  required ImagePickerPort picker,
  required PermissionPort perms,
  required UploadApi uploads,
  required MediaCache media,
}) {
  return ProviderContainer(
    overrides: [
      ...chatTestOverrides(repo),
      imagePickerPortProvider.overrideWithValue(picker),
      imageCompressorPortProvider.overrideWithValue(_FakeCompressor()),
      permissionPortProvider.overrideWithValue(perms),
      uploadApiProvider.overrideWithValue(uploads),
      mediaCacheProvider.overrideWithValue(media),
    ],
  );
}

Future<String> _tempImage(String name) async {
  final file = File('${Directory.systemTemp.path}/sg_test_$name.jpg');
  await file.writeAsBytes([1, 2, 3, 4]);
  return file.path;
}

void main() {
  test('camera → compress → upload → send forwards the storage key', () async {
    final path = await _tempImage('cam');
    final repo = FakeChatRepository();
    final uploads = _FakeUploadApi();
    final media = _FakeMediaCache();
    final container = _container(
      repo: repo,
      picker: _FakePicker(cameraPath: path),
      perms: _FakePermissions(PermissionOutcome.granted),
      uploads: uploads,
      media: media,
    );
    addTearDown(container.dispose);
    await container.read(chatControllerProvider.future);
    final composer = container.read(messageComposerProvider.notifier);

    final added = await composer.addFromCamera();
    expect(added, AttachRequestResult.added);
    expect(container.read(messageComposerProvider).attachments, hasLength(1));

    final sent = await composer.uploadAndSend('look');
    await pumpEventQueue();

    expect(sent, isTrue);
    expect(uploads.putCalls, 1);
    expect(media.stored, ['u/a/img/1.jpg']);
    expect(repo.lastSendText, 'look');
    expect(repo.lastSendImageKeys, ['u/a/img/1.jpg']);
    // staging cleared after dispatch
    expect(container.read(messageComposerProvider).attachments, isEmpty);
  });

  test('gallery selection is capped at the per-message photo limit', () async {
    final repo = FakeChatRepository();
    final picker = _FakePicker(
      galleryPaths: List.generate(6, (i) => '/tmp/g$i.jpg'),
    );
    final container = _container(
      repo: repo,
      picker: picker,
      perms: _FakePermissions(PermissionOutcome.granted),
      uploads: _FakeUploadApi(),
      media: _FakeMediaCache(),
    );
    addTearDown(container.dispose);
    final composer = container.read(messageComposerProvider.notifier);

    await composer.addFromGallery();

    expect(picker.lastGalleryLimit, kMaxPhotosPerMessage);
    expect(
      container.read(messageComposerProvider).attachments,
      hasLength(kMaxPhotosPerMessage),
    );
    expect(container.read(messageComposerProvider).canAddMore, isFalse);
  });

  test('denied permission returns a blocked result and stages nothing', () async {
    final repo = FakeChatRepository();
    final container = _container(
      repo: repo,
      picker: _FakePicker(cameraPath: '/tmp/x.jpg'),
      perms: _FakePermissions(PermissionOutcome.denied),
      uploads: _FakeUploadApi(),
      media: _FakeMediaCache(),
    );
    addTearDown(container.dispose);
    final composer = container.read(messageComposerProvider.notifier);

    final result = await composer.addFromCamera();

    expect(result, AttachRequestResult.permissionDenied);
    expect(container.read(messageComposerProvider).attachments, isEmpty);
  });

  test('a failed upload keeps staging and does not send', () async {
    final path = await _tempImage('fail');
    final repo = FakeChatRepository();
    final container = _container(
      repo: repo,
      picker: _FakePicker(cameraPath: path),
      perms: _FakePermissions(PermissionOutcome.granted),
      uploads: _FakeUploadApi(fail: true),
      media: _FakeMediaCache(),
    );
    addTearDown(container.dispose);
    await container.read(chatControllerProvider.future);
    final composer = container.read(messageComposerProvider.notifier);

    await composer.addFromCamera();
    final sent = await composer.uploadAndSend('look');
    await pumpEventQueue();

    expect(sent, isFalse);
    expect(repo.lastSendText, isNull); // sendMessage never called
    final state = container.read(messageComposerProvider);
    expect(state.uploading, isFalse);
    expect(state.attachments, hasLength(1));
    expect(state.attachments.single.status, AttachStatus.failed);
  });
}
