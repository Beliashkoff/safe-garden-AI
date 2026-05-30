import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../data/media_cache.dart';

/// Renders the photos of a message bubble. Each [storageKeys] entry is resolved
/// to a local file via [mediaFileProvider] (cache hit for a just-sent photo,
/// lazy download for history); failures fall back to a placeholder.
class MessagePhotos extends ConsumerWidget {
  const MessagePhotos({required this.storageKeys, super.key});

  final List<String> storageKeys;

  static const double _size = 140;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Wrap(
      spacing: 4,
      runSpacing: 4,
      children: [for (final key in storageKeys) _thumb(context, ref, key)],
    );
  }

  Widget _thumb(BuildContext context, WidgetRef ref, String storageKey) {
    final async = ref.watch(mediaFileProvider(storageKey));
    return ClipRRect(
      borderRadius: BorderRadius.circular(10),
      child: SizedBox(
        width: _size,
        height: _size,
        child: async.when(
          data: (file) => Image.file(
            file,
            width: _size,
            height: _size,
            fit: BoxFit.cover,
            errorBuilder: (context, _, _) => _placeholder(context, broken: true),
          ),
          loading: () => _placeholder(context, broken: false),
          error: (_, _) => _placeholder(context, broken: true),
        ),
      ),
    );
  }

  Widget _placeholder(BuildContext context, {required bool broken}) {
    final theme = Theme.of(context);
    return Container(
      color: theme.colorScheme.surfaceContainerHighest,
      alignment: Alignment.center,
      child: broken
          ? Icon(Icons.broken_image_outlined, color: theme.colorScheme.outline)
          : const SizedBox(
              height: 22,
              width: 22,
              child: CircularProgressIndicator(strokeWidth: 2),
            ),
    );
  }
}
