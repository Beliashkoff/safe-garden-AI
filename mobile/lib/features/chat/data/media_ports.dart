import 'dart:io';

import 'package:flutter_image_compress/flutter_image_compress.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:path_provider/path_provider.dart';
import 'package:permission_handler/permission_handler.dart';

/// Outcome of a runtime permission request, collapsed to the three states the
/// composer reacts to.
enum PermissionOutcome { granted, denied, permanentlyDenied }

/// Selects images from the camera or gallery. Wraps `image_picker` behind a
/// narrow port so the composer can be unit-tested without the platform plugin.
abstract interface class ImagePickerPort {
  /// Captures a single photo; returns its file path, or null if cancelled.
  Future<String?> pickCamera();

  /// Picks up to [limit] photos from the gallery; returns their file paths.
  Future<List<String>> pickGallery({required int limit});
}

/// Compresses a source image to the project's photo budget (1920×1080, q=85,
/// JPEG) and returns the resulting file.
abstract interface class ImageCompressorPort {
  Future<File> compress(String sourcePath);
}

/// Requests camera / photo-library permissions and opens app settings.
abstract interface class PermissionPort {
  Future<PermissionOutcome> ensureCamera();
  Future<PermissionOutcome> ensurePhotos();
  Future<void> openSettings();
}

/// `image_picker`-backed implementation. For a single gallery photo it uses
/// `pickImage` (some platforms require `pickMultiImage`'s limit to be ≥ 2).
class SystemImagePicker implements ImagePickerPort {
  SystemImagePicker([ImagePicker? picker]) : _picker = picker ?? ImagePicker();

  final ImagePicker _picker;

  @override
  Future<String?> pickCamera() async {
    final file = await _picker.pickImage(source: ImageSource.camera);
    return file?.path;
  }

  @override
  Future<List<String>> pickGallery({required int limit}) async {
    if (limit <= 1) {
      final file = await _picker.pickImage(source: ImageSource.gallery);
      return file == null ? <String>[] : [file.path];
    }
    final files = await _picker.pickMultiImage(limit: limit);
    return files.map((f) => f.path).toList();
  }
}

/// `flutter_image_compress`-backed implementation. Always outputs JPEG so the
/// presign content type is uniform and HEIC (iOS) is normalized.
class FlutterImageCompressor implements ImageCompressorPort {
  const FlutterImageCompressor();

  @override
  Future<File> compress(String sourcePath) async {
    final dir = await getTemporaryDirectory();
    final target =
        '${dir.path}/sg_${DateTime.now().microsecondsSinceEpoch}.jpg';
    final result = await FlutterImageCompress.compressAndGetFile(
      sourcePath,
      target,
      minWidth: 1920,
      minHeight: 1080,
      quality: 85,
      format: CompressFormat.jpeg,
    );
    if (result == null) {
      throw const FileSystemException('image compression failed');
    }
    return File(result.path);
  }
}

/// `permission_handler`-backed implementation. On Android the gallery uses the
/// system Photo Picker, which needs no runtime permission, so [ensurePhotos]
/// short-circuits there; the camera always needs permission.
class SystemPermissions implements PermissionPort {
  const SystemPermissions();

  @override
  Future<PermissionOutcome> ensureCamera() =>
      _request(Permission.camera);

  @override
  Future<PermissionOutcome> ensurePhotos() async {
    if (Platform.isAndroid) {
      return PermissionOutcome.granted;
    }
    return _request(Permission.photos);
  }

  @override
  Future<void> openSettings() => openAppSettings();

  Future<PermissionOutcome> _request(Permission permission) async {
    final status = await permission.request();
    if (status.isGranted || status.isLimited) {
      return PermissionOutcome.granted;
    }
    if (status.isPermanentlyDenied || status.isRestricted) {
      return PermissionOutcome.permanentlyDenied;
    }
    return PermissionOutcome.denied;
  }
}

final imagePickerPortProvider = Provider<ImagePickerPort>(
  (ref) => SystemImagePicker(),
);

final imageCompressorPortProvider = Provider<ImageCompressorPort>(
  (ref) => const FlutterImageCompressor(),
);

final permissionPortProvider = Provider<PermissionPort>(
  (ref) => const SystemPermissions(),
);
