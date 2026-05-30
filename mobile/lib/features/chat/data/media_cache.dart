import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path_provider/path_provider.dart';

import 'upload_api.dart';

/// Resolves an image `storage_key` to a local file. A photo the user just sent
/// is stored here right after upload, so its bubble renders from disk without a
/// round-trip; a photo from history that is not cached locally (reinstall, other
/// device) is fetched lazily via the presigned view URL and then cached.
abstract interface class MediaCache {
  /// Returns the cached file for [storageKey], or null if not present locally.
  Future<File?> fileForKey(String storageKey);

  /// Copies [source] into the cache under [storageKey] and returns the cached
  /// file. Called after a successful upload.
  Future<File> store(String storageKey, File source);

  /// Returns the cached file for [storageKey], downloading and caching it first
  /// when missing. Throws [AppException] on a failed fetch.
  Future<File> ensure(String storageKey);

  /// Removes all cached media (e.g. on logout, to honour the ownership
  /// invariant — a different user on the device must not see prior photos).
  Future<void> clear();
}

class FileMediaCache implements MediaCache {
  FileMediaCache(this._uploads);

  final UploadApi _uploads;
  Directory? _dir;

  Future<Directory> _mediaDir() async {
    final cached = _dir;
    if (cached != null) {
      return cached;
    }
    final base = await getApplicationDocumentsDirectory();
    final dir = Directory('${base.path}/media');
    if (!await dir.exists()) {
      await dir.create(recursive: true);
    }
    _dir = dir;
    return dir;
  }

  Future<File> _pathFor(String storageKey) async {
    final dir = await _mediaDir();
    final name = sha256.convert(utf8.encode(storageKey)).toString();
    return File('${dir.path}/$name.jpg');
  }

  @override
  Future<File?> fileForKey(String storageKey) async {
    final file = await _pathFor(storageKey);
    return await file.exists() ? file : null;
  }

  @override
  Future<File> store(String storageKey, File source) async {
    final dest = await _pathFor(storageKey);
    return source.copy(dest.path);
  }

  @override
  Future<File> ensure(String storageKey) async {
    final existing = await fileForKey(storageKey);
    if (existing != null) {
      return existing;
    }
    final view = await _uploads.presignView(storageKey: storageKey);
    final bytes = await _uploads.downloadBytes(url: view.url);
    final dest = await _pathFor(storageKey);
    return dest.writeAsBytes(bytes, flush: true);
  }

  @override
  Future<void> clear() async {
    final dir = await _mediaDir();
    if (await dir.exists()) {
      await dir.delete(recursive: true);
    }
    _dir = null;
  }
}

final mediaCacheProvider = Provider<MediaCache>(
  (ref) => FileMediaCache(ref.watch(uploadApiProvider)),
);

/// Resolves the local file for an image `storage_key`, caching the result so a
/// bubble does not re-fetch on every rebuild. Errors surface as the provider's
/// error state, which the photo widget renders as a placeholder.
final mediaFileProvider = FutureProvider.family<File, String>(
  (ref, storageKey) => ref.watch(mediaCacheProvider).ensure(storageKey),
);
