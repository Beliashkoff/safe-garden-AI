import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_exception.dart';
import '../data/media_cache.dart';
import '../data/media_ports.dart';
import '../data/upload_api.dart';
import 'chat_controller.dart';

/// Maximum photos per message (SPEC F4 / ARCH §6.1).
const int kMaxPhotosPerMessage = 4;

/// Lifecycle of a staged photo as it moves from picked → uploaded.
enum AttachStatus { ready, uploading, uploaded, failed }

/// The result the composer returns to the UI after an attach attempt, so the
/// widget layer (which owns BuildContext) can show the right prompt.
enum AttachRequestResult {
  added,
  cancelled,
  limitReached,
  permissionDenied,
  permissionPermanentlyDenied,
  failed,
}

/// A photo staged in the composer: its compressed local [file], upload [status]
/// and [progress] (0..1), and the [storageKey] assigned once uploaded.
class PhotoAttachment {
  const PhotoAttachment({
    required this.localId,
    required this.file,
    this.status = AttachStatus.ready,
    this.progress = 0,
    this.storageKey,
  });

  final String localId;
  final File file;
  final AttachStatus status;
  final double progress;
  final String? storageKey;

  PhotoAttachment copyWith({
    AttachStatus? status,
    double? progress,
    String? storageKey,
  }) {
    return PhotoAttachment(
      localId: localId,
      file: file,
      status: status ?? this.status,
      progress: progress ?? this.progress,
      storageKey: storageKey ?? this.storageKey,
    );
  }
}

/// Immutable composer state: the staged [attachments] and whether an upload is
/// in flight.
class ComposerState {
  const ComposerState({this.attachments = const [], this.uploading = false});

  final List<PhotoAttachment> attachments;
  final bool uploading;

  bool get hasAttachments => attachments.isNotEmpty;
  bool get canAddMore => attachments.length < kMaxPhotosPerMessage;

  ComposerState copyWith({
    List<PhotoAttachment>? attachments,
    bool? uploading,
  }) {
    return ComposerState(
      attachments: attachments ?? this.attachments,
      uploading: uploading ?? this.uploading,
    );
  }
}

/// Drives the photo-attachment composer: permission → pick → compress →
/// stage, then on send: presign → PUT (with progress) → cache, and finally
/// hands the storage keys to [ChatController.sendMessage]. The actual messaging
/// stays in the chat controller; this notifier owns only the upload pipeline.
class MessageComposer extends Notifier<ComposerState> {
  @override
  ComposerState build() => const ComposerState();

  ImagePickerPort get _picker => ref.read(imagePickerPortProvider);
  ImageCompressorPort get _compressor => ref.read(imageCompressorPortProvider);
  PermissionPort get _perms => ref.read(permissionPortProvider);
  UploadApi get _uploads => ref.read(uploadApiProvider);
  MediaCache get _media => ref.read(mediaCacheProvider);

  int _seq = 0;

  Future<AttachRequestResult> addFromCamera() async {
    if (!state.canAddMore) {
      return AttachRequestResult.limitReached;
    }
    final outcome = await _perms.ensureCamera();
    final blocked = _outcomeToResult(outcome);
    if (blocked != null) {
      return blocked;
    }
    final path = await _picker.pickCamera();
    if (path == null) {
      return AttachRequestResult.cancelled;
    }
    return _ingest([path]);
  }

  Future<AttachRequestResult> addFromGallery() async {
    if (!state.canAddMore) {
      return AttachRequestResult.limitReached;
    }
    final outcome = await _perms.ensurePhotos();
    final blocked = _outcomeToResult(outcome);
    if (blocked != null) {
      return blocked;
    }
    final remaining = kMaxPhotosPerMessage - state.attachments.length;
    final paths = await _picker.pickGallery(limit: remaining);
    if (paths.isEmpty) {
      return AttachRequestResult.cancelled;
    }
    return _ingest(paths.take(remaining).toList());
  }

  /// Compresses each picked path and stages it. Stops at the photo limit.
  Future<AttachRequestResult> _ingest(List<String> paths) async {
    try {
      for (final path in paths) {
        if (!state.canAddMore) {
          break;
        }
        final file = await _compressor.compress(path);
        state = state.copyWith(
          attachments: [
            ...state.attachments,
            PhotoAttachment(localId: _newId(), file: file),
          ],
        );
      }
      return AttachRequestResult.added;
    } on Object {
      return AttachRequestResult.failed;
    }
  }

  void remove(String localId) {
    state = state.copyWith(
      attachments:
          state.attachments.where((a) => a.localId != localId).toList(),
    );
  }

  void clear() => state = const ComposerState();

  /// Uploads every staged photo (presign → PUT → cache) then dispatches the
  /// message via the chat controller. Returns false (leaving staging intact, so
  /// the user can retry) if any upload fails. Caller passes the input [text].
  Future<bool> uploadAndSend(String text) async {
    if (!state.hasAttachments || state.uploading) {
      return false;
    }
    state = state.copyWith(uploading: true);

    final keys = <String>[];
    for (final att in state.attachments) {
      final key = await _uploadOne(att);
      if (key == null) {
        state = state.copyWith(uploading: false);
        return false;
      }
      keys.add(key);
    }

    clear();
    await ref
        .read(chatControllerProvider.notifier)
        .sendMessage(text, imageStorageKeys: keys);
    return true;
  }

  Future<String?> _uploadOne(PhotoAttachment att) async {
    _patch(att.localId, (a) => a.copyWith(status: AttachStatus.uploading, progress: 0));
    try {
      final bytes = await att.file.readAsBytes();
      final presign = await _uploads.presignPut(
        contentType: 'image/jpeg',
        sizeBytes: bytes.length,
      );
      await _uploads.putObject(
        url: presign.url,
        headers: presign.headers,
        bytes: bytes,
        onSendProgress: (sent, total) {
          if (total > 0) {
            _patch(att.localId, (a) => a.copyWith(progress: sent / total));
          }
        },
      );
      await _media.store(presign.key, att.file);
      _patch(
        att.localId,
        (a) => a.copyWith(
          status: AttachStatus.uploaded,
          progress: 1,
          storageKey: presign.key,
        ),
      );
      return presign.key;
    } on AppException {
      _patch(att.localId, (a) => a.copyWith(status: AttachStatus.failed));
      return null;
    }
  }

  void _patch(String localId, PhotoAttachment Function(PhotoAttachment) update) {
    state = state.copyWith(
      attachments: [
        for (final a in state.attachments)
          if (a.localId == localId) update(a) else a,
      ],
    );
  }

  AttachRequestResult? _outcomeToResult(PermissionOutcome outcome) {
    switch (outcome) {
      case PermissionOutcome.granted:
        return null;
      case PermissionOutcome.denied:
        return AttachRequestResult.permissionDenied;
      case PermissionOutcome.permanentlyDenied:
        return AttachRequestResult.permissionPermanentlyDenied;
    }
  }

  String _newId() =>
      'att-${DateTime.now().microsecondsSinceEpoch}-${_seq++}';
}

final messageComposerProvider =
    NotifierProvider<MessageComposer, ComposerState>(MessageComposer.new);
